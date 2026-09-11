package registry

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TenantModuleStatus represents tenant module activation status
type TenantModuleStatus string

// The activation states of a module installed for a tenant.
const (
	TenantModuleStatusActive       TenantModuleStatus = "active"
	TenantModuleStatusInactive     TenantModuleStatus = "inactive"
	TenantModuleStatusSuspended    TenantModuleStatus = "suspended"
	TenantModuleStatusPendingSetup TenantModuleStatus = "pending_setup"
)

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

// The billing states of a tenant's subscription to a module or a tool.
const (
	SubscriptionStatusTrial     SubscriptionStatus = "trial"
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusCanceled  SubscriptionStatus = "canceled"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusSuspended SubscriptionStatus = "suspended"
)

// TenantModule represents a module activation for a tenant
type TenantModule struct {
	ID                      string             `json:"id"`
	TenantID                string             `json:"tenant_id"`
	ModuleID                string             `json:"module_id"`
	Status                  TenantModuleStatus `json:"status"`
	IsEnabled               bool               `json:"is_enabled"`
	InstalledAt             time.Time          `json:"installed_at"`
	ActivatedAt             *time.Time         `json:"activated_at,omitempty"`
	DeactivatedAt           *time.Time         `json:"deactivated_at,omitempty"`
	LastUsedAt              *time.Time         `json:"last_used_at,omitempty"`
	InstalledVersion        string             `json:"installed_version"`
	LatestCompatibleVersion string             `json:"latest_compatible_version,omitempty"`
	Configuration           json.RawMessage    `json:"configuration"`
	FeaturesEnabled         json.RawMessage    `json:"features_enabled"`
	Limits                  json.RawMessage    `json:"limits"`
	CurrentUsage            json.RawMessage    `json:"current_usage"`
	SubscriptionStatus      SubscriptionStatus `json:"subscription_status,omitempty"`
	SubscriptionStart       *time.Time         `json:"subscription_start,omitempty"`
	SubscriptionEnd         *time.Time         `json:"subscription_end,omitempty"`
	TrialEndsAt             *time.Time         `json:"trial_ends_at,omitempty"`
	NextBillingDate         *time.Time         `json:"next_billing_date,omitempty"`
	PricingPlan             string             `json:"pricing_plan,omitempty"`
	BillingCycle            string             `json:"billing_cycle,omitempty"`
	AmountPaid              float64            `json:"amount_paid,omitempty"`
	Currency                string             `json:"currency,omitempty"`
	SetupCompleted          bool               `json:"setup_completed"`
	SetupStepsCompleted     json.RawMessage    `json:"setup_steps_completed"`
	OnboardingCompleted     bool               `json:"onboarding_completed"`
	AllowedRoles            []string           `json:"allowed_roles,omitempty"`
	RestrictedFeatures      json.RawMessage    `json:"restricted_features"`
	Notes                   string             `json:"notes,omitempty"`
	Metadata                json.RawMessage    `json:"metadata"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
	DeletedAt               *time.Time         `json:"deleted_at,omitempty"`
	CreatedBy               string             `json:"created_by,omitempty"`
	UpdatedBy               string             `json:"updated_by,omitempty"`
	ActivatedBy             string             `json:"activated_by,omitempty"`
	DeactivatedBy           string             `json:"deactivated_by,omitempty"`
}

// NewTenantModule creates a new tenant module activation
func NewTenantModule(tenantID, moduleID, version string) (*TenantModule, error) {
	now := time.Now()

	tm := &TenantModule{
		ID:                  uuid.New().String(),
		TenantID:            tenantID,
		ModuleID:            moduleID,
		Status:              TenantModuleStatusPendingSetup,
		IsEnabled:           false,
		InstalledAt:         now,
		InstalledVersion:    version,
		SetupCompleted:      false,
		OnboardingCompleted: false,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := tm.Validate(); err != nil {
		return nil, err
	}

	return tm, nil
}

// Validate validates tenant module data
func (tm *TenantModule) Validate() error {
	if tm.ID == "" {
		return ErrInvalidTenantModuleID
	}
	if tm.TenantID == "" {
		return ErrTenantIDRequired
	}
	if tm.ModuleID == "" {
		return ErrModuleIDRequired
	}
	if !tm.Status.IsValid() {
		return ErrInvalidTenantModuleStatus
	}
	if tm.SubscriptionStatus != "" && !tm.SubscriptionStatus.IsValid() {
		return ErrInvalidSubscriptionStatus
	}

	return nil
}

// Activate activates the tenant module
func (tm *TenantModule) Activate(activatedBy string) error {
	if tm.Status == TenantModuleStatusActive {
		return ErrTenantModuleAlreadyActive
	}

	now := time.Now()
	tm.Status = TenantModuleStatusActive
	tm.IsEnabled = true
	tm.ActivatedAt = &now
	tm.ActivatedBy = activatedBy
	tm.UpdatedAt = now

	return nil
}

// Deactivate deactivates the tenant module
func (tm *TenantModule) Deactivate(deactivatedBy string) error {
	if tm.Status == TenantModuleStatusInactive {
		return ErrTenantModuleAlreadyInactive
	}

	now := time.Now()
	tm.Status = TenantModuleStatusInactive
	tm.IsEnabled = false
	tm.DeactivatedAt = &now
	tm.DeactivatedBy = deactivatedBy
	tm.UpdatedAt = now

	return nil
}

// Suspend suspends the tenant module
func (tm *TenantModule) Suspend() {
	tm.Status = TenantModuleStatusSuspended
	tm.IsEnabled = false
	tm.UpdatedAt = time.Now()
}

// CompleteSetup marks setup as completed
func (tm *TenantModule) CompleteSetup() {
	tm.SetupCompleted = true
	if tm.Status == TenantModuleStatusPendingSetup {
		tm.Status = TenantModuleStatusActive
		now := time.Now()
		tm.ActivatedAt = &now
	}
	tm.UpdatedAt = time.Now()
}

// CompleteOnboarding marks onboarding as completed
func (tm *TenantModule) CompleteOnboarding() {
	tm.OnboardingCompleted = true
	tm.UpdatedAt = time.Now()
}

// UpdateLastUsed updates last used timestamp
func (tm *TenantModule) UpdateLastUsed() {
	now := time.Now()
	tm.LastUsedAt = &now
	tm.UpdatedAt = now
}

// IsActive checks if tenant module is active
func (tm *TenantModule) IsActive() bool {
	return tm.Status == TenantModuleStatusActive && tm.IsEnabled
}

// IsValid checks if tenant module status is valid
func (s TenantModuleStatus) IsValid() bool {
	switch s {
	case TenantModuleStatusActive, TenantModuleStatusInactive,
		TenantModuleStatusSuspended, TenantModuleStatusPendingSetup:
		return true
	default:
		return false
	}
}

// IsValid checks if subscription status is valid
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case SubscriptionStatusTrial, SubscriptionStatusActive,
		SubscriptionStatusCanceled, SubscriptionStatusPastDue,
		SubscriptionStatusSuspended:
		return true
	default:
		return false
	}
}
