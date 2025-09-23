package tenant

import (
	"context"
	"errors"
	"fmt"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/subscription"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
	"nexspaces-api/internal/core/ports/events"
	"nexspaces-api/internal/core/ports/repositories"
	"nexspaces-api/internal/core/ports/services"
)

// CreateTenantUseCase handles tenant creation
type CreateTenantUseCase struct {
	tenantRepo     repositories.TenantRepository
	userRepo       repositories.UserRepository
	planRepo       repositories.PlanRepository
	eventBus       events.EventBus
	billingService services.BillingService
	notifyService  services.NotificationService
	logger         services.Logger
}

// NewCreateTenantUseCase creates a new CreateTenantUseCase
func NewCreateTenantUseCase(
	tenantRepo repositories.TenantRepository,
	userRepo repositories.UserRepository,
	planRepo repositories.PlanRepository,
	eventBus events.EventBus,
	billingService services.BillingService,
	notifyService services.NotificationService,
	logger services.Logger,
) *CreateTenantUseCase {
	return &CreateTenantUseCase{
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		planRepo:       planRepo,
		eventBus:       eventBus,
		billingService: billingService,
		notifyService:  notifyService,
		logger:         logger,
	}
}

// CreateTenantRequest represents the request to create a tenant
type CreateTenantRequest struct {
	Name         string                 `json:"name" validate:"required,min=2,max=100"`
	Slug         string                 `json:"slug" validate:"required,min=2,max=63"`
	CustomDomain *string                `json:"custom_domain,omitempty" validate:"omitempty,fqdn"`
	PlanID       *string                `json:"plan_id,omitempty" validate:"omitempty,uuid"`
	Settings     shared.Settings        `json:"settings,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`

	// Owner information
	OwnerEmail    string `json:"owner_email" validate:"required,email"`
	OwnerName     string `json:"owner_name" validate:"required,min=2,max=255"`
	OwnerPassword string `json:"owner_password" validate:"required,min=8"`

	// Billing information (optional for trial)
	PaymentMethodID *string `json:"payment_method_id,omitempty"`
	StartTrial      bool    `json:"start_trial"`
	CouponCode      *string `json:"coupon_code,omitempty"`
}

// CreateTenantResponse represents the response from creating a tenant
type CreateTenantResponse struct {
	Tenant        *TenantResponse       `json:"tenant"`
	Owner         *UserResponse         `json:"owner"`
	Subscription  *SubscriptionResponse `json:"subscription,omitempty"`
	AuthTokens    *services.TokenPair   `json:"auth_tokens"`
	OnboardingURL string                `json:"onboarding_url"`
}

// TenantResponse represents tenant response data
type TenantResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	CustomDomain *string                `json:"custom_domain"`
	Status       string                 `json:"status"`
	PlanID       *string                `json:"plan_id"`
	Settings     shared.Settings        `json:"settings"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    shared.Timestamp       `json:"created_at"`
	UpdatedAt    shared.Timestamp       `json:"updated_at"`
}

// UserResponse represents user response data
type UserResponse struct {
	ID        string           `json:"id"`
	TenantID  string           `json:"tenant_id"`
	Email     string           `json:"email"`
	Name      string           `json:"name"`
	Role      string           `json:"role"`
	Status    string           `json:"status"`
	CreatedAt shared.Timestamp `json:"created_at"`
	UpdatedAt shared.Timestamp `json:"updated_at"`
}

// SubscriptionResponse represents subscription response data
type SubscriptionResponse struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	PlanID          string            `json:"plan_id"`
	Status          string            `json:"status"`
	BillingCycle    string            `json:"billing_cycle"`
	NextBillingDate shared.Timestamp  `json:"next_billing_date"`
	TrialEndsAt     *shared.Timestamp `json:"trial_ends_at"`
	CreatedAt       shared.Timestamp  `json:"created_at"`
}

// Execute executes the create tenant use case
func (uc *CreateTenantUseCase) Execute(ctx context.Context, req CreateTenantRequest) (*CreateTenantResponse, error) {
	// Log the operation
	uc.logger.Info(ctx, "Creating tenant", map[string]interface{}{
		"tenant_name": req.Name,
		"tenant_slug": req.Slug,
		"owner_email": req.OwnerEmail,
		"start_trial": req.StartTrial,
	})

	// Validate request
	if err := uc.validateRequest(ctx, req); err != nil {
		uc.logger.Error(ctx, "Tenant creation validation failed", err, map[string]interface{}{
			"tenant_slug": req.Slug,
			"owner_email": req.OwnerEmail,
		})
		return nil, err
	}

	// Check if slug is available
	exists, err := uc.tenantRepo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check tenant slug availability", err, map[string]interface{}{
			"tenant_slug": req.Slug,
		})
		return nil, fmt.Errorf("failed to check slug availability: %w", err)
	}
	if exists {
		return nil, shared.NewConflictError("tenant slug already exists")
	}

	// Get plan if specified
	var planID *string
	if req.PlanID != nil {
		plan, err := uc.planRepo.GetByID(ctx, shared.ParseIDMust(*req.PlanID))
		if err != nil {
			uc.logger.Error(ctx, "Failed to get plan", err, map[string]interface{}{
				"plan_id": *req.PlanID,
			})
			return nil, fmt.Errorf("invalid plan ID: %w", err)
		}
		if !plan.IsActive() {
			return nil, shared.NewValidationError("selected plan is not active")
		}
		planID = req.PlanID
	} else {
		// Get default plan
		defaultPlan, err := uc.planRepo.GetDefaultPlan(ctx)
		if err != nil {
			uc.logger.Error(ctx, "Failed to get default plan", err, nil)
			return nil, fmt.Errorf("failed to get default plan: %w", err)
		}
		planIDStr := defaultPlan.ID().String()
		planID = &planIDStr
	}

	// Create tenant domain entity
	newTenant, err := tenant.NewTenant(tenant.TenantParams{
		Name:         req.Name,
		Slug:         req.Slug,
		CustomDomain: req.CustomDomain,
		PlanID:       planID,
		Settings:     req.Settings,
		Metadata:     req.Metadata,
	})
	if err != nil {
		uc.logger.Error(ctx, "Failed to create tenant domain entity", err, map[string]interface{}{
			"tenant_name": req.Name,
			"tenant_slug": req.Slug,
		})
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Save tenant to repository
	if err := uc.tenantRepo.Create(ctx, newTenant); err != nil {
		uc.logger.Error(ctx, "Failed to save tenant to repository", err, map[string]interface{}{
			"tenant_id":   newTenant.ID().String(),
			"tenant_name": newTenant.Name(),
		})
		return nil, fmt.Errorf("failed to save tenant: %w", err)
	}

	// Create owner user
	owner, err := uc.createOwnerUser(ctx, newTenant.ID(), req)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create owner user", err, map[string]interface{}{
			"tenant_id":   newTenant.ID().String(),
			"owner_email": req.OwnerEmail,
		})

		// Rollback tenant creation
		if rollbackErr := uc.tenantRepo.Delete(ctx, newTenant.ID()); rollbackErr != nil {
			uc.logger.Error(ctx, "Failed to rollback tenant creation", rollbackErr, map[string]interface{}{
				"tenant_id": newTenant.ID().String(),
			})
		}
		return nil, fmt.Errorf("failed to create owner user: %w", err)
	}

	// Create subscription
	subscription, err := uc.createSubscription(ctx, newTenant.ID(), *planID, req)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create subscription", err, map[string]interface{}{
			"tenant_id": newTenant.ID().String(),
			"plan_id":   *planID,
		})

		// Note: We might want to continue without subscription for some cases
		// or implement proper compensation pattern
		uc.logger.Warn(ctx, "Continuing without subscription due to error", map[string]interface{}{
			"tenant_id": newTenant.ID().String(),
			"error":     err.Error(),
		})
	}

	// Generate auth tokens for owner
	authTokens, err := uc.generateAuthTokens(ctx, owner.ID(), newTenant.ID())
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate auth tokens", err, map[string]interface{}{
			"tenant_id": newTenant.ID().String(),
			"user_id":   owner.ID().String(),
		})
		// Don't fail the whole operation for auth tokens
		uc.logger.Warn(ctx, "Continuing without auth tokens", map[string]interface{}{
			"tenant_id": newTenant.ID().String(),
		})
	}

	// Publish domain events
	if err := uc.publishEvents(ctx, newTenant, owner, subscription); err != nil {
		uc.logger.Error(ctx, "Failed to publish domain events", err, map[string]interface{}{
			"tenant_id": newTenant.ID().String(),
		})
		// Don't fail for event publishing
	}

	// Send welcome notifications
	go uc.sendWelcomeNotifications(context.Background(), newTenant, owner)

	// Prepare response
	response := &CreateTenantResponse{
		Tenant: &TenantResponse{
			ID:           newTenant.ID().String(),
			Name:         newTenant.Name(),
			Slug:         newTenant.Slug().String(),
			CustomDomain: newTenant.CustomDomain(),
			Status:       newTenant.Status().String(),
			PlanID:       newTenant.PlanID(),
			Settings:     newTenant.Settings(),
			Metadata:     newTenant.Metadata(),
			CreatedAt:    newTenant.CreatedAt(),
			UpdatedAt:    newTenant.UpdatedAt(),
		},
		Owner: &UserResponse{
			ID:        owner.ID().String(),
			TenantID:  owner.TenantID().String(),
			Email:     owner.Email().String(),
			Name:      owner.Name(),
			Role:      owner.Role().String(),
			Status:    owner.Status().String(),
			CreatedAt: owner.CreatedAt(),
			UpdatedAt: owner.UpdatedAt(),
		},
		OnboardingURL: fmt.Sprintf("https://%s.nexpaces.com/onboarding", newTenant.Slug().String()),
	}

	if subscription != nil {
		response.Subscription = &SubscriptionResponse{
			ID:              subscription.ID().String(),
			TenantID:        subscription.TenantID().String(),
			PlanID:          subscription.PlanID().String(),
			Status:          subscription.Status().String(),
			BillingCycle:    subscription.BillingCycle().String(),
			NextBillingDate: subscription.NextBillingDate(),
			TrialEndsAt:     subscription.TrialEndsAt(),
			CreatedAt:       subscription.CreatedAt(),
		}
	}

	if authTokens != nil {
		response.AuthTokens = authTokens
	}

	uc.logger.Info(ctx, "Tenant created successfully", map[string]interface{}{
		"tenant_id":        newTenant.ID().String(),
		"tenant_name":      newTenant.Name(),
		"owner_email":      owner.Email().String(),
		"has_subscription": subscription != nil,
		"has_auth_tokens":  authTokens != nil,
	})

	return response, nil
}

// validateRequest validates the create tenant request
func (uc *CreateTenantUseCase) validateRequest(ctx context.Context, req CreateTenantRequest) error {
	// Basic validation is handled by struct tags, this is for business rules

	// Check if custom domain is already in use (if provided)
	if req.CustomDomain != nil && *req.CustomDomain != "" {
		existing, err := uc.tenantRepo.GetByCustomDomain(ctx, *req.CustomDomain)
		if err != nil && !shared.IsNotFoundError(err) {
			return fmt.Errorf("failed to check custom domain: %w", err)
		}
		if existing != nil {
			return shared.NewConflictError("custom domain already in use")
		}
	}

	// Check if owner email is already in use
	existingUsers, err := uc.userRepo.GetUserAcrossTenantsForAuth(ctx, req.OwnerEmail)
	if err != nil {
		return fmt.Errorf("failed to check owner email: %w", err)
	}
	if len(existingUsers) > 0 {
		return shared.NewConflictError("email address already in use")
	}

	// Validate payment method if provided
	if req.PaymentMethodID != nil && !req.StartTrial {
		// This would validate with billing service
		// For now, we assume it's valid if provided
	}

	return nil
}

// createOwnerUser creates the owner user for the tenant
func (uc *CreateTenantUseCase) createOwnerUser(ctx context.Context, tenantID tenant.TenantID, req CreateTenantRequest) (*user.User, error) {
	// Create user domain entity
	owner, err := user.NewUser(user.UserParams{
		TenantID: tenantID,
		Email:    req.OwnerEmail,
		Name:     req.OwnerName,
		Role:     user.RoleOwner,
		Metadata: make(map[string]interface{}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create owner user entity: %w", err)
	}

	// Save user to repository
	if err := uc.userRepo.Create(ctx, owner); err != nil {
		return nil, fmt.Errorf("failed to save owner user: %w", err)
	}

	return owner, nil
}

// createSubscription creates subscription for the tenant
func (uc *CreateTenantUseCase) createSubscription(ctx context.Context, tenantID tenant.TenantID, planID string, req CreateTenantRequest) (*subscription.Subscription, error) {
	// Create subscription through billing service
	subscriptionReq := services.CreateSubscriptionRequest{
		TenantID:        tenantID,
		PlanID:          shared.ParseIDMust(planID),
		BillingCycle:    subscription.CycleMonthly, // Default to monthly
		PaymentMethodID: req.PaymentMethodID,
		StartTrial:      req.StartTrial,
		CouponCode:      req.CouponCode,
		Metadata: map[string]interface{}{
			"created_during_tenant_setup": true,
		},
	}

	if req.PaymentMethodID == nil && !req.StartTrial {
		// No payment method and no trial - might be a free plan
		// Let billing service handle this logic
	}

	sub, err := uc.billingService.CreateSubscription(ctx, subscriptionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub, nil
}

// generateAuthTokens generates authentication tokens for the owner
func (uc *CreateTenantUseCase) generateAuthTokens(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) (*services.TokenPair, error) {
	// This would be implemented by auth service
	// For now, return nil to indicate no auth tokens generated
	return nil, nil
}

// publishEvents publishes domain events
func (uc *CreateTenantUseCase) publishEvents(ctx context.Context, tenant *tenant.Tenant, owner *user.User, subscription *subscription.Subscription) error {
	var events []events.DomainEvent

	// Tenant created event
	tenantEvent := events.NewTenantCreatedEvent(
		tenant.ID(),
		tenant.Name(),
		tenant.Slug().String(),
		tenant.PlanID(),
		owner.ID(),
	)
	events = append(events, tenantEvent)

	// User created event
	userEvent := events.NewUserCreatedEvent(
		owner.ID(),
		owner.TenantID(),
		owner.Email().String(),
		owner.Name(),
		owner.Role(),
	)
	events = append(events, userEvent)

	// Subscription created event (if exists)
	if subscription != nil {
		subscriptionEvent := subscription.GenerateCreatedEvent()
		events = append(events, &subscriptionEvent)
	}

	// Publish events in batch
	return uc.eventBus.PublishBatch(ctx, events)
}

// sendWelcomeNotifications sends welcome notifications
func (uc *CreateTenantUseCase) sendWelcomeNotifications(ctx context.Context, tenant *tenant.Tenant, owner *user.User) {
	// Send welcome email to owner
	welcomeReq := services.SendTemplatedEmailRequest{
		TenantID:   tenant.ID(),
		TemplateID: "welcome_owner",
		To:         []string{owner.Email().String()},
		TemplateData: map[string]interface{}{
			"owner_name":    owner.Name(),
			"tenant_name":   tenant.Name(),
			"tenant_slug":   tenant.Slug().String(),
			"dashboard_url": fmt.Sprintf("https://%s.nexpaces.com/dashboard", tenant.Slug().String()),
		},
	}

	if err := uc.notifyService.SendTemplatedEmail(ctx, welcomeReq); err != nil {
		uc.logger.Error(ctx, "Failed to send welcome email", err, map[string]interface{}{
			"tenant_id":   tenant.ID().String(),
			"owner_email": owner.Email().String(),
		})
	}

	// Send in-app notification
	inAppReq := services.SendInAppNotificationRequest{
		TenantID:   tenant.ID(),
		UserID:     owner.ID(),
		Type:       services.InAppTypeSuccess,
		Title:      "Welcome to NexSpaces!",
		Message:    fmt.Sprintf("Your workspace '%s' has been created successfully. Start by exploring templates in our marketplace.", tenant.Name()),
		ActionURL:  "/templates/marketplace",
		ActionText: "Explore Templates",
		Priority:   services.PriorityNormal,
	}

	if err := uc.notifyService.SendInAppNotification(ctx, inAppReq); err != nil {
		uc.logger.Error(ctx, "Failed to send in-app notification", err, map[string]interface{}{
			"tenant_id": tenant.ID().String(),
			"user_id":   owner.ID().String(),
		})
	}
}

// Helper function to parse ID (assuming this exists in shared package)
func ParseIDMust(s string) shared.ID {
	id, err := shared.ParseID(s)
	if err != nil {
		panic(fmt.Sprintf("invalid ID: %s", s))
	}
	return id
}

// Add to shared package
func IsNotFoundError(err error) bool {
	var domainErr *shared.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == shared.NotFoundError
	}
	return false
}
