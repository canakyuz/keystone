package subscription

import (
	"github.com/google/uuid"
	"time"
)

// Subscription represents a tenant's subscription
type Subscription struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	TenantID        uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	PlanID          string       `json:"plan_id" db:"plan_id"`
	Status          Status       `json:"status" db:"status"`
	BillingCycle    BillingCycle `json:"billing_cycle" db:"billing_cycle"`
	StartDate       time.Time    `json:"start_date" db:"start_date"`
	EndDate         *time.Time   `json:"end_date,omitempty" db:"end_date"`
	TrialEndDate    *time.Time   `json:"trial_end_date,omitempty" db:"trial_end_date"`
	UsageLimits     UsageLimits  `json:"usage_limits" db:"usage_limits"`
	CurrentUsage    CurrentUsage `json:"current_usage" db:"current_usage"`
	Amount          float64      `json:"amount" db:"amount"`
	Currency        string       `json:"currency" db:"currency"`
	StripeID        string       `json:"stripe_id,omitempty" db:"stripe_subscription_id"`
	ExternalID      string       `json:"external_id,omitempty" db:"external_id"`
	PaymentMethodID string       `json:"payment_method_id,omitempty" db:"payment_method_id"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at" db:"updated_at"`
}

// Activate activates the subscription.
func (s *Subscription) Activate() error {
	s.Status = StatusActive
	s.UpdatedAt = time.Now()
	return nil
}

// Status represents subscription status
type Status string

const (
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
	StatusSuspended Status = "suspended"
	StatusTrial     Status = "trial"
	StatusExpired   Status = "expired"
)

// BillingCycle represents billing frequency
type BillingCycle string

const (
	BillingMonthly BillingCycle = "monthly"
	BillingYearly  BillingCycle = "yearly"
)

// UsageLimits represents limits for the subscription plan
type UsageLimits struct {
	Templates     int `json:"templates"`
	Users         int `json:"users"`
	Storage       int `json:"storage_gb"`
	APIRequests   int `json:"api_requests_per_month"`
	CustomDomains int `json:"custom_domains"`
}

// CurrentUsage represents current usage for the subscription
type CurrentUsage struct {
	Templates     int `json:"templates"`
	Users         int `json:"users"`
	Storage       int `json:"storage_gb"`
	APIRequests   int `json:"api_requests_this_month"`
	CustomDomains int `json:"custom_domains"`
}

// CreateSubscriptionRequest represents the data needed to create a new subscription
type CreateSubscriptionRequest struct {
	TenantID     uuid.UUID    `json:"tenant_id" validate:"required"`
	PlanID       string       `json:"plan_id" validate:"required"`
	BillingCycle BillingCycle `json:"billing_cycle" validate:"required,oneof=monthly yearly"`
	TrialDays    *int         `json:"trial_days,omitempty"`
}

// UpdateSubscriptionRequest represents the data that can be updated for a subscription
type UpdateSubscriptionRequest struct {
	PlanID       *string       `json:"plan_id,omitempty"`
	BillingCycle *BillingCycle `json:"billing_cycle,omitempty" validate:"omitempty,oneof=monthly yearly"`
	Status       *Status       `json:"status,omitempty" validate:"omitempty,oneof=active cancelled suspended trial expired"`
}

// SubscriptionResponse represents the subscription data returned in API responses
type SubscriptionResponse struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     uuid.UUID    `json:"tenant_id"`
	PlanID       string       `json:"plan_id"`
	Status       Status       `json:"status"`
	BillingCycle BillingCycle `json:"billing_cycle"`
	StartDate    time.Time    `json:"start_date"`
	EndDate      *time.Time   `json:"end_date,omitempty"`
	TrialEndDate *time.Time   `json:"trial_end_date,omitempty"`
	UsageLimits  UsageLimits  `json:"usage_limits"`
	CurrentUsage CurrentUsage `json:"current_usage"`
	Amount       float64      `json:"amount"`
	Currency     string       `json:"currency"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ToResponse converts a Subscription domain object to a SubscriptionResponse
func (s *Subscription) ToResponse() *SubscriptionResponse {
	return &SubscriptionResponse{
		ID:           s.ID,
		TenantID:     s.TenantID,
		PlanID:       s.PlanID,
		Status:       s.Status,
		BillingCycle: s.BillingCycle,
		StartDate:    s.StartDate,
		EndDate:      s.EndDate,
		TrialEndDate: s.TrialEndDate,
		UsageLimits:  s.UsageLimits,
		CurrentUsage: s.CurrentUsage,
		Amount:       s.Amount,
		Currency:     s.Currency,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// IsActive checks if the subscription is currently active
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive || s.Status == StatusTrial
}

// IsInTrial checks if the subscription is in trial period
func (s *Subscription) IsInTrial() bool {
	if s.Status != StatusTrial || s.TrialEndDate == nil {
		return false
	}
	return time.Now().Before(*s.TrialEndDate)
}

// HasExceededLimit checks if current usage has exceeded the limit for a specific resource
func (s *Subscription) HasExceededLimit(resource string) bool {
	switch resource {
	case "templates":
		return s.CurrentUsage.Templates >= s.UsageLimits.Templates
	case "users":
		return s.CurrentUsage.Users >= s.UsageLimits.Users
	case "storage":
		return s.CurrentUsage.Storage >= s.UsageLimits.Storage
	case "api_requests":
		return s.CurrentUsage.APIRequests >= s.UsageLimits.APIRequests
	case "custom_domains":
		return s.CurrentUsage.CustomDomains >= s.UsageLimits.CustomDomains
	}
	return false
}

// GetUsagePercentage returns the usage percentage for a specific resource
func (s *Subscription) GetUsagePercentage(resource string) float64 {
	switch resource {
	case "templates":
		if s.UsageLimits.Templates == 0 {
			return 0
		}
		return (float64(s.CurrentUsage.Templates) / float64(s.UsageLimits.Templates)) * 100
	case "users":
		if s.UsageLimits.Users == 0 {
			return 0
		}
		return (float64(s.CurrentUsage.Users) / float64(s.UsageLimits.Users)) * 100
	case "storage":
		if s.UsageLimits.Storage == 0 {
			return 0
		}
		return (float64(s.CurrentUsage.Storage) / float64(s.UsageLimits.Storage)) * 100
	case "api_requests":
		if s.UsageLimits.APIRequests == 0 {
			return 0
		}
		return (float64(s.CurrentUsage.APIRequests) / float64(s.UsageLimits.APIRequests)) * 100
	case "custom_domains":
		if s.UsageLimits.CustomDomains == 0 {
			return 0
		}
		return (float64(s.CurrentUsage.CustomDomains) / float64(s.UsageLimits.CustomDomains)) * 100
	}
	return 0
}

// NewSubscription creates a new Subscription instance
func NewSubscription(req CreateSubscriptionRequest, plan Plan) *Subscription {
	now := time.Now()

	subscription := &Subscription{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		PlanID:       req.PlanID,
		Status:       StatusActive,
		BillingCycle: req.BillingCycle,
		StartDate:    now,
		UsageLimits:  plan.Limits,
		CurrentUsage: CurrentUsage{}, // Initialize with zero usage
		Amount:       plan.GetPrice(req.BillingCycle),
		Currency:     plan.Currency,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Set trial period if specified
	if req.TrialDays != nil && *req.TrialDays > 0 {
		subscription.Status = StatusTrial
		trialEnd := now.AddDate(0, 0, *req.TrialDays)
		subscription.TrialEndDate = &trialEnd
	}

	return subscription
}

// Plan represents a subscription plan
type Plan struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Limits       UsageLimits `json:"limits"`
	MonthlyPrice float64     `json:"monthly_price"`
	YearlyPrice  float64     `json:"yearly_price"`
	Currency     string      `json:"currency"`
	Features     []string    `json:"features"`
	IsPopular    bool        `json:"is_popular"`
}

// GetPrice returns the price for the specified billing cycle
func (p *Plan) GetPrice(cycle BillingCycle) float64 {
	if cycle == BillingYearly {
		return p.YearlyPrice
	}
	return p.MonthlyPrice
}
