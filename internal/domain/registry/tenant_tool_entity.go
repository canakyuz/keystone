package registry

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TenantToolStatus represents tenant tool activation status
type TenantToolStatus string

const (
	TenantToolStatusActive              TenantToolStatus = "active"
	TenantToolStatusInactive            TenantToolStatus = "inactive"
	TenantToolStatusSuspended           TenantToolStatus = "suspended"
	TenantToolStatusPendingSetup        TenantToolStatus = "pending_setup"
	TenantToolStatusPendingVerification TenantToolStatus = "pending_verification"
)

// IntegrationStatus represents external integration status
type IntegrationStatus string

const (
	IntegrationStatusConnected    IntegrationStatus = "connected"
	IntegrationStatusDisconnected IntegrationStatus = "disconnected"
	IntegrationStatusError        IntegrationStatus = "error"
	IntegrationStatusPending      IntegrationStatus = "pending"
)

// HealthStatus represents tool health status
type HealthStatus string

const (
	HealthStatusHealthy     HealthStatus = "healthy"
	HealthStatusDegraded    HealthStatus = "degraded"
	HealthStatusUnavailable HealthStatus = "unavailable"
)

// TenantTool represents a tool activation for a tenant
type TenantTool struct {
	ID                       string             `json:"id"`
	TenantID                 string             `json:"tenant_id"`
	ToolID                   string             `json:"tool_id"`
	ModuleID                 string             `json:"module_id,omitempty"` // Optional: if activated via module
	Status                   TenantToolStatus   `json:"status"`
	IsEnabled                bool               `json:"is_enabled"`
	InstalledAt              time.Time          `json:"installed_at"`
	ActivatedAt              *time.Time         `json:"activated_at,omitempty"`
	DeactivatedAt            *time.Time         `json:"deactivated_at,omitempty"`
	LastUsedAt               *time.Time         `json:"last_used_at,omitempty"`
	InstalledVersion         string             `json:"installed_version"`
	LatestCompatibleVersion  string             `json:"latest_compatible_version,omitempty"`
	Configuration            json.RawMessage    `json:"configuration"`
	APIKeys                  json.RawMessage    `json:"api_keys"` // Encrypted
	WebhookConfig            json.RawMessage    `json:"webhook_config"`
	IntegrationEnabled       bool               `json:"integration_enabled"`
	IntegrationStatus        IntegrationStatus  `json:"integration_status,omitempty"`
	IntegrationVerified      bool               `json:"integration_verified"`
	IntegrationVerifiedAt    *time.Time         `json:"integration_verified_at,omitempty"`
	ProviderAccountID        string             `json:"provider_account_id,omitempty"`
	Limits                   json.RawMessage    `json:"limits"`
	CurrentUsage             json.RawMessage    `json:"current_usage"`
	RateLimits               json.RawMessage    `json:"rate_limits"`
	SubscriptionStatus       SubscriptionStatus `json:"subscription_status,omitempty"`
	SubscriptionStart        *time.Time         `json:"subscription_start,omitempty"`
	SubscriptionEnd          *time.Time         `json:"subscription_end,omitempty"`
	TrialEndsAt              *time.Time         `json:"trial_ends_at,omitempty"`
	NextBillingDate          *time.Time         `json:"next_billing_date,omitempty"`
	PricingPlan              string             `json:"pricing_plan,omitempty"`
	BillingCycle             string             `json:"billing_cycle,omitempty"`
	AmountPaid               float64            `json:"amount_paid,omitempty"`
	Currency                 string             `json:"currency,omitempty"`
	TransactionFeesCollected float64            `json:"transaction_fees_collected,omitempty"`
	SetupCompleted           bool               `json:"setup_completed"`
	SetupStepsCompleted      json.RawMessage    `json:"setup_steps_completed"`
	OnboardingCompleted      bool               `json:"onboarding_completed"`
	AllowedRoles             []string           `json:"allowed_roles,omitempty"`
	RestrictedFeatures       json.RawMessage    `json:"restricted_features"`
	HealthStatus             HealthStatus       `json:"health_status"`
	LastHealthCheck          *time.Time         `json:"last_health_check,omitempty"`
	ErrorCount               int                `json:"error_count"`
	LastError                string             `json:"last_error,omitempty"`
	LastErrorAt              *time.Time         `json:"last_error_at,omitempty"`
	Notes                    string             `json:"notes,omitempty"`
	Metadata                 json.RawMessage    `json:"metadata"`
	CreatedAt                time.Time          `json:"created_at"`
	UpdatedAt                time.Time          `json:"updated_at"`
	DeletedAt                *time.Time         `json:"deleted_at,omitempty"`
	CreatedBy                string             `json:"created_by,omitempty"`
	UpdatedBy                string             `json:"updated_by,omitempty"`
	ActivatedBy              string             `json:"activated_by,omitempty"`
	DeactivatedBy            string             `json:"deactivated_by,omitempty"`
}

// NewTenantTool creates a new tenant tool activation
func NewTenantTool(tenantID, toolID, version string) (*TenantTool, error) {
	now := time.Now()

	tt := &TenantTool{
		ID:                  uuid.New().String(),
		TenantID:            tenantID,
		ToolID:              toolID,
		Status:              TenantToolStatusPendingSetup,
		IsEnabled:           false,
		InstalledAt:         now,
		InstalledVersion:    version,
		IntegrationEnabled:  false,
		IntegrationVerified: false,
		SetupCompleted:      false,
		OnboardingCompleted: false,
		HealthStatus:        HealthStatusHealthy,
		ErrorCount:          0,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := tt.Validate(); err != nil {
		return nil, err
	}

	return tt, nil
}

// Validate validates tenant tool data
func (tt *TenantTool) Validate() error {
	if tt.ID == "" {
		return ErrInvalidTenantToolID
	}
	if tt.TenantID == "" {
		return ErrTenantIDRequired
	}
	if tt.ToolID == "" {
		return ErrToolIDRequired
	}
	if !tt.Status.IsValid() {
		return ErrInvalidTenantToolStatus
	}
	if tt.IntegrationStatus != "" && !tt.IntegrationStatus.IsValid() {
		return ErrInvalidIntegrationStatus
	}
	if !tt.HealthStatus.IsValid() {
		return ErrInvalidHealthStatus
	}
	if tt.SubscriptionStatus != "" && !tt.SubscriptionStatus.IsValid() {
		return ErrInvalidSubscriptionStatus
	}

	return nil
}

// Activate activates the tenant tool
func (tt *TenantTool) Activate(activatedBy string) error {
	if tt.Status == TenantToolStatusActive {
		return ErrTenantToolAlreadyActive
	}

	now := time.Now()
	tt.Status = TenantToolStatusActive
	tt.IsEnabled = true
	tt.ActivatedAt = &now
	tt.ActivatedBy = activatedBy
	tt.UpdatedAt = now

	return nil
}

// Deactivate deactivates the tenant tool
func (tt *TenantTool) Deactivate(deactivatedBy string) error {
	if tt.Status == TenantToolStatusInactive {
		return ErrTenantToolAlreadyInactive
	}

	now := time.Now()
	tt.Status = TenantToolStatusInactive
	tt.IsEnabled = false
	tt.DeactivatedAt = &now
	tt.DeactivatedBy = deactivatedBy
	tt.UpdatedAt = now

	return nil
}

// VerifyIntegration marks integration as verified
func (tt *TenantTool) VerifyIntegration() {
	now := time.Now()
	tt.IntegrationVerified = true
	tt.IntegrationVerifiedAt = &now
	tt.IntegrationStatus = IntegrationStatusConnected
	tt.UpdatedAt = now
}

// SetHealthStatus sets health status
func (tt *TenantTool) SetHealthStatus(status HealthStatus) {
	tt.HealthStatus = status
	now := time.Now()
	tt.LastHealthCheck = &now
	tt.UpdatedAt = now
}

// RecordError records an error
func (tt *TenantTool) RecordError(errorMsg string) {
	tt.ErrorCount++
	tt.LastError = errorMsg
	now := time.Now()
	tt.LastErrorAt = &now
	tt.UpdatedAt = now

	// Degrade health if too many errors
	if tt.ErrorCount > 10 {
		tt.HealthStatus = HealthStatusDegraded
	}
	if tt.ErrorCount > 50 {
		tt.HealthStatus = HealthStatusUnavailable
	}
}

// ResetErrorCount resets the error count
func (tt *TenantTool) ResetErrorCount() {
	tt.ErrorCount = 0
	tt.LastError = ""
	tt.LastErrorAt = nil
	tt.HealthStatus = HealthStatusHealthy
	tt.UpdatedAt = time.Now()
}

// CompleteSetup marks setup as completed
func (tt *TenantTool) CompleteSetup() {
	tt.SetupCompleted = true
	if tt.Status == TenantToolStatusPendingSetup {
		tt.Status = TenantToolStatusActive
		now := time.Now()
		tt.ActivatedAt = &now
	}
	tt.UpdatedAt = time.Now()
}

// UpdateLastUsed updates last used timestamp
func (tt *TenantTool) UpdateLastUsed() {
	now := time.Now()
	tt.LastUsedAt = &now
	tt.UpdatedAt = now
}

// IsActive checks if tenant tool is active
func (tt *TenantTool) IsActive() bool {
	return tt.Status == TenantToolStatusActive && tt.IsEnabled
}

// IsHealthy checks if tool is healthy
func (tt *TenantTool) IsHealthy() bool {
	return tt.HealthStatus == HealthStatusHealthy
}

// IsValid checks if tenant tool status is valid
func (s TenantToolStatus) IsValid() bool {
	switch s {
	case TenantToolStatusActive, TenantToolStatusInactive,
		TenantToolStatusSuspended, TenantToolStatusPendingSetup,
		TenantToolStatusPendingVerification:
		return true
	default:
		return false
	}
}

// IsValid checks if integration status is valid
func (s IntegrationStatus) IsValid() bool {
	switch s {
	case IntegrationStatusConnected, IntegrationStatusDisconnected,
		IntegrationStatusError, IntegrationStatusPending:
		return true
	default:
		return false
	}
}

// IsValid checks if health status is valid
func (s HealthStatus) IsValid() bool {
	switch s {
	case HealthStatusHealthy, HealthStatusDegraded, HealthStatusUnavailable:
		return true
	default:
		return false
	}
}
