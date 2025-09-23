package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/subscription"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
	"nexspaces-api/internal/core/ports/events"
	"nexspaces-api/internal/core/ports/repositories"
	"nexspaces-api/internal/core/ports/services"
)

// CreateSubscriptionUseCase handles subscription creation for tenants
type CreateSubscriptionUseCase struct {
	subscriptionRepo repositories.SubscriptionRepository
	tenantRepo       repositories.TenantRepository
	userRepo         repositories.UserRepository
	billingService   services.BillingService
	eventBus         events.EventBus
}

func NewCreateSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	tenantRepo repositories.TenantRepository,
	userRepo repositories.UserRepository,
	billingService services.BillingService,
	eventBus events.EventBus,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		userRepo:         userRepo,
		billingService:   billingService,
		eventBus:         eventBus,
	}
}

type CreateSubscriptionRequest struct {
	TenantID        shared.TenantID           `json:"tenant_id" validate:"required"`
	PlanID          shared.PlanID             `json:"plan_id" validate:"required"`
	BillingCycle    subscription.BillingCycle `json:"billing_cycle" validate:"required"`
	PaymentMethodID string                    `json:"payment_method_id" validate:"required"`
	CreatedBy       shared.UserID             `json:"created_by" validate:"required"`
}

type CreateSubscriptionResponse struct {
	Subscription  *subscription.Subscription `json:"subscription"`
	PaymentIntent string                     `json:"payment_intent,omitempty"`
}

func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, req CreateSubscriptionRequest) (*CreateSubscriptionResponse, error) {
	// Validate tenant exists
	tenantEntity, err := uc.tenantRepo.GetByID(ctx, uuid.UUID(req.TenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Validate creator
	creator, err := uc.userRepo.GetByID(ctx, uuid.UUID(req.TenantID), uuid.UUID(req.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("failed to get creator: %w", err)
	}

	// Verify tenant isolation
	if creator.TenantID != uuid.UUID(req.TenantID) {
		return nil, shared.ErrCrossTenantAccess
	}

	// Check permissions (only admins can manage subscriptions)
	if !creator.CanManageBilling() {
		return nil, shared.ErrInsufficientPermissions
	}

	// Check if tenant already has active subscription
	existingSubscription, err := uc.subscriptionRepo.GetActiveByTenant(ctx, uuid.UUID(req.TenantID))
	if err != nil && !errors.Is(err, shared.ErrSubscriptionNotFound) {
		return nil, fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if existingSubscription != nil {
		return nil, shared.ErrTenantAlreadyHasSubscription
	}

	// Get plan details from billing service
	plan, err := uc.billingService.GetPlan(ctx, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Create subscription entity
	createReq := subscription.CreateSubscriptionRequest{
		TenantID:     uuid.UUID(req.TenantID),
		PlanID:       string(req.PlanID),
		BillingCycle: req.BillingCycle,
	}
	planEntity := subscription.Plan{
		ID:           string(plan.ID),
		MonthlyPrice: plan.Price,
		YearlyPrice:  plan.Price, // Assuming yearly is same as monthly for now
		Currency:     plan.Currency,
		Features:     plan.Features,
		Limits:       subscription.UsageLimits{},
	}
	subscriptionEntity := subscription.NewSubscription(createReq, planEntity)

	// Process payment through billing service
	paymentResult, err := uc.billingService.ProcessSubscriptionPayment(ctx, services.SubscriptionPaymentRequest{
		TenantID:        req.TenantID,
		SubscriptionID:  shared.SubscriptionID(subscriptionEntity.ID),
		PlanID:          req.PlanID,
		Amount:          plan.Price,
		Currency:        plan.Currency,
		BillingCycle:    string(req.BillingCycle),
		PaymentMethodID: req.PaymentMethodID,
	})
	if err != nil {
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// Update subscription with payment info
	subscriptionEntity.ExternalID = paymentResult.SubscriptionID
	subscriptionEntity.PaymentMethodID = req.PaymentMethodID

	// Activate subscription if payment succeeded
	if paymentResult.Status == "active" {
		if err := subscriptionEntity.Activate(); err != nil {
			return nil, fmt.Errorf("failed to activate subscription: %w", err)
		}
	}

	// Save subscription
	if err := uc.subscriptionRepo.Create(ctx, subscriptionEntity); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Update tenant subscription reference
	tenantEntity.SubscriptionID = &subscriptionEntity.ID
	if err := uc.tenantRepo.Update(ctx, tenantEntity); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, events.SubscriptionCreatedEvent{
		SubscriptionID: subscriptionEntity.ID,
		TenantID:       req.TenantID,
		PlanID:         req.PlanID,
		BillingCycle:   req.BillingCycle,
		Amount:         plan.Price,
		Currency:       plan.Currency,
		CreatedBy:      req.CreatedBy,
		CreatedAt:      subscriptionEntity.CreatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return &CreateSubscriptionResponse{
		Subscription:  subscriptionEntity,
		PaymentIntent: paymentResult.PaymentIntentID,
	}, nil
}

// UpdateSubscriptionUseCase handles subscription plan changes
type UpdateSubscriptionUseCase struct {
	subscriptionRepo repositories.SubscriptionRepository
	tenantRepo       repositories.TenantRepository
	userRepo         repositories.UserRepository
	billingService   services.BillingService
	eventBus         events.EventBus
}

func NewUpdateSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	tenantRepo repositories.TenantRepository,
	userRepo repositories.UserRepository,
	billingService services.BillingService,
	eventBus events.EventBus,
) *UpdateSubscriptionUseCase {
	return &UpdateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		userRepo:         userRepo,
		billingService:   billingService,
		eventBus:         eventBus,
	}
}

type UpdateSubscriptionRequest struct {
	SubscriptionID shared.SubscriptionID `json:"subscription_id" validate:"required"`
	TenantID       shared.TenantID       `json:"tenant_id" validate:"required"`
	NewPlanID      shared.PlanID         `json:"new_plan_id" validate:"required"`
	UpdatedBy      shared.UserID         `json:"updated_by" validate:"required"`
	ProrationMode  string                `json:"proration_mode,omitempty"` // immediate, next_billing_period
}

func (uc *UpdateSubscriptionUseCase) Execute(ctx context.Context, req UpdateSubscriptionRequest) (*subscription.Subscription, error) {
	// Get subscription
	subscriptionEntity, err := uc.subscriptionRepo.GetByID(ctx, req.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Verify tenant isolation
	if subscriptionEntity.TenantID != req.TenantID {
		return nil, shared.ErrCrossTenantAccess
	}

	// Validate updater
	updater, err := uc.userRepo.GetByID(ctx, req.TenantID, req.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get updater: %w", err)
	}

	if updater.TenantID != req.TenantID {
		return nil, shared.ErrCrossTenantAccess
	}

	// Check permissions
	if !updater.CanManageBilling() {
		return nil, shared.ErrInsufficientPermissions
	}

	// Get new plan details
	newPlan, err := uc.billingService.GetPlan(ctx, req.NewPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new plan: %w", err)
	}

	// Update subscription through billing service
	updateResult, err := uc.billingService.UpdateSubscription(ctx, services.UpdateSubscriptionRequest{
		SubscriptionID: subscriptionEntity.ExternalID,
		NewPlanID:      req.NewPlanID,
		ProrationMode:  req.ProrationMode,
	})
	if err != nil {
		return nil, fmt.Errorf("billing service update failed: %w", err)
	}

	// Update subscription entity
	oldPlanID := subscriptionEntity.PlanID
	if err := subscriptionEntity.UpdatePlan(req.NewPlanID, newPlan.Price, newPlan.Features, newPlan.Limits); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Save changes
	if err := uc.subscriptionRepo.Update(ctx, subscriptionEntity); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, events.SubscriptionUpdatedEvent{
		SubscriptionID: subscriptionEntity.ID,
		TenantID:       req.TenantID,
		OldPlanID:      oldPlanID,
		NewPlanID:      req.NewPlanID,
		ProrationMode:  req.ProrationMode,
		UpdatedBy:      req.UpdatedBy,
		UpdatedAt:      subscriptionEntity.UpdatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return subscriptionEntity, nil
}

// CancelSubscriptionUseCase handles subscription cancellation
type CancelSubscriptionUseCase struct {
	subscriptionRepo repositories.SubscriptionRepository
	userRepo         repositories.UserRepository
	billingService   services.BillingService
	eventBus         events.EventBus
}

func NewCancelSubscriptionUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	userRepo repositories.UserRepository,
	billingService services.BillingService,
	eventBus events.EventBus,
) *CancelSubscriptionUseCase {
	return &CancelSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		billingService:   billingService,
		eventBus:         eventBus,
	}
}

type CancelSubscriptionRequest struct {
	SubscriptionID shared.SubscriptionID `json:"subscription_id" validate:"required"`
	TenantID       shared.TenantID       `json:"tenant_id" validate:"required"`
	CancelledBy    shared.UserID         `json:"cancelled_by" validate:"required"`
	CancelMode     string                `json:"cancel_mode" validate:"required"` // immediate, end_of_period
	Reason         string                `json:"reason,omitempty"`
}

func (uc *CancelSubscriptionUseCase) Execute(ctx context.Context, req CancelSubscriptionRequest) error {
	// Get subscription
	subscriptionEntity, err := uc.subscriptionRepo.GetByID(ctx, req.SubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Verify tenant isolation
	if subscriptionEntity.TenantID != req.TenantID {
		return shared.ErrCrossTenantAccess
	}

	// Validate canceller
	canceller, err := uc.userRepo.GetByID(ctx, req.TenantID, req.CancelledBy)
	if err != nil {
		return fmt.Errorf("failed to get canceller: %w", err)
	}

	if canceller.TenantID != req.TenantID {
		return shared.ErrCrossTenantAccess
	}

	// Check permissions
	if !canceller.CanManageBilling() {
		return shared.ErrInsufficientPermissions
	}

	// Cancel through billing service
	cancelResult, err := uc.billingService.CancelSubscription(ctx, services.CancelSubscriptionRequest{
		SubscriptionID: subscriptionEntity.ExternalID,
		CancelMode:     req.CancelMode,
		Reason:         req.Reason,
	})
	if err != nil {
		return fmt.Errorf("billing service cancellation failed: %w", err)
	}

	// Update subscription entity
	if err := subscriptionEntity.Cancel(req.Reason, cancelResult.EffectiveDate); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	// Save changes
	if err := uc.subscriptionRepo.Update(ctx, subscriptionEntity); err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, events.SubscriptionCancelledEvent{
		SubscriptionID: subscriptionEntity.ID,
		TenantID:       req.TenantID,
		CancelledBy:    req.CancelledBy,
		CancelMode:     req.CancelMode,
		Reason:         req.Reason,
		EffectiveDate:  cancelResult.EffectiveDate,
		CancelledAt:    subscriptionEntity.UpdatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return nil
}

// TrackUsageUseCase handles usage tracking for subscription limits
type TrackUsageUseCase struct {
	subscriptionRepo repositories.SubscriptionRepository
	tenantRepo       repositories.TenantRepository
	eventBus         events.EventBus
}

func NewTrackUsageUseCase(
	subscriptionRepo repositories.SubscriptionRepository,
	tenantRepo repositories.TenantRepository,
	eventBus events.EventBus,
) *TrackUsageUseCase {
	return &TrackUsageUseCase{
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		eventBus:         eventBus,
	}
}

type TrackUsageRequest struct {
	TenantID   shared.TenantID  `json:"tenant_id" validate:"required"`
	MetricName string           `json:"metric_name" validate:"required"`
	Usage      int64            `json:"usage" validate:"min=1"`
	Timestamp  shared.Timestamp `json:"timestamp"`
}

type TrackUsageResponse struct {
	RemainingUsage int64 `json:"remaining_usage"`
	LimitExceeded  bool  `json:"limit_exceeded"`
}

func (uc *TrackUsageUseCase) Execute(ctx context.Context, req TrackUsageRequest) (*TrackUsageResponse, error) {
	// Get active subscription for tenant
	subscriptionEntity, err := uc.subscriptionRepo.GetActiveByTenant(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Track usage
	remainingUsage, limitExceeded, err := subscriptionEntity.TrackUsage(req.MetricName, req.Usage, req.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to track usage: %w", err)
	}

	// Save updated subscription
	if err := uc.subscriptionRepo.Update(ctx, subscriptionEntity); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Publish usage event
	if err := uc.eventBus.Publish(ctx, events.UsageTrackedEvent{
		SubscriptionID: subscriptionEntity.ID,
		TenantID:       req.TenantID,
		MetricName:     req.MetricName,
		Usage:          req.Usage,
		RemainingUsage: remainingUsage,
		LimitExceeded:  limitExceeded,
		Timestamp:      req.Timestamp,
	}); err != nil {
		// Log error but don't fail
	}

	// Publish limit exceeded event if needed
	if limitExceeded {
		if err := uc.eventBus.Publish(ctx, events.UsageLimitExceededEvent{
			SubscriptionID: subscriptionEntity.ID,
			TenantID:       req.TenantID,
			MetricName:     req.MetricName,
			CurrentUsage:   subscriptionEntity.CurrentUsage[req.MetricName],
			Limit:          subscriptionEntity.UsageLimits[req.MetricName],
			Timestamp:      req.Timestamp,
		}); err != nil {
			// Log error but don't fail
		}
	}

	return &TrackUsageResponse{
		RemainingUsage: remainingUsage,
		LimitExceeded:  limitExceeded,
	}, nil
}
