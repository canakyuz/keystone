package tenant

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the status of a tenant account
type TenantStatus string

const (
	// TenantStatusPending: the record exists but provisioning has not started.
	TenantStatusPending TenantStatus = "pending"
	// TenantStatusProvisioning: the worker is preparing the schema.
	// A tenant in this state accepts no requests: its schema is not ready.
	TenantStatusProvisioning TenantStatus = "provisioning"
	// TenantStatusFailed: provisioning failed.
	// It is distinct from TenantStatusInactive, which means a working tenant was shut
	// down. The distinction answers the question "can provisioning be retried?"
	TenantStatusFailed TenantStatus = "failed"

	// TenantStatusActive serves traffic. It and TenantStatusTrial are the only states the
	// tenant schema cache resolves; a request for a tenant in any other state is refused.
	TenantStatusActive TenantStatus = "active"
	// TenantStatusSuspended: a working tenant was stopped, and can be reactivated.
	TenantStatusSuspended TenantStatus = "suspended"
	// TenantStatusInactive: a working tenant was shut down.
	TenantStatusInactive TenantStatus = "inactive"
	// TenantStatusTrial serves traffic as TenantStatusActive does, on a trial.
	TenantStatusTrial TenantStatus = "trial"
)

// SubscriptionPlan represents different subscription tiers
type SubscriptionPlan string

// The subscription plans. A tenant's plan sets its rate limit quota and picks the schema
// template applied when it is provisioned.
const (
	PlanFree       SubscriptionPlan = "free"
	PlanStarter    SubscriptionPlan = "starter"
	PlanPro        SubscriptionPlan = "pro"
	PlanEnterprise SubscriptionPlan = "enterprise"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Slug       string       `json:"slug"` // URL-friendly identifier
	Email      string       `json:"email"`
	Phone      string       `json:"phone,omitempty"`
	SchemaName string       `json:"schema_name"`
	Status     TenantStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`

	// Subscription
	Plan              SubscriptionPlan `json:"plan"`
	SubscriptionStart *time.Time       `json:"subscription_start,omitempty"`
	SubscriptionEnd   *time.Time       `json:"subscription_end,omitempty"`
	TrialEndsAt       *time.Time       `json:"trial_ends_at,omitempty"`

	// Custom domain
	CustomDomain           string     `json:"custom_domain,omitempty"`
	CustomDomainVerified   bool       `json:"custom_domain_verified"`
	CustomDomainVerifiedAt *time.Time `json:"custom_domain_verified_at,omitempty"`

	// Metadata
	Settings map[string]any `json:"settings,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`

	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// New creates a new tenant with default values
func New(name, slug, email string, plan SubscriptionPlan) (*Tenant, error) {
	now := time.Now()

	tenant := &Tenant{
		ID:                   uuid.New().String(),
		Name:                 name,
		Slug:                 slug,
		Email:                email,
		Status:               TenantStatusActive,
		Plan:                 plan,
		CreatedAt:            now,
		UpdatedAt:            now,
		CustomDomainVerified: false,
		Settings:             make(map[string]any),
		Metadata:             make(map[string]any),
	}

	// Set trial period for free/starter plans
	if plan == PlanFree || plan == PlanStarter {
		trialEnd := now.AddDate(0, 0, 14) // 14 days trial
		tenant.TrialEndsAt = &trialEnd
		tenant.Status = TenantStatusTrial
	}

	// Validate tenant
	if err := tenant.Validate(); err != nil {
		return nil, err
	}

	return tenant, nil
}

// Validate validates tenant data
func (t *Tenant) Validate() error {
	if t.ID == "" {
		return ErrInvalidTenantID
	}

	if t.Name == "" {
		return ErrTenantNameRequired
	}

	if len(t.Name) < 2 || len(t.Name) > 100 {
		return ErrInvalidTenantName
	}

	if t.Slug == "" {
		return ErrTenantSlugRequired
	}

	if len(t.Slug) < 2 || len(t.Slug) > 50 {
		return ErrInvalidTenantSlug
	}

	if t.Email == "" {
		return ErrTenantEmailRequired
	}

	if !t.Status.IsValid() {
		return ErrInvalidTenantStatus
	}

	if !t.Plan.IsValid() {
		return ErrInvalidSubscriptionPlan
	}

	if t.SchemaName != "" {
		if err := ValidateSchemaName(t.SchemaName); err != nil {
			return err
		}
	}

	return nil
}

var schemaNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// ValidateSchemaName ensures tenant schemas comply with PostgreSQL identifier rules.
func ValidateSchemaName(name string) error {
	if name == "" {
		return ErrTenantSchemaNameRequired
	}

	if !schemaNamePattern.MatchString(name) {
		return ErrInvalidTenantSchemaName
	}

	return nil
}

// SetSchemaName assigns the generated schema name to the tenant entity
func (t *Tenant) SetSchemaName(name string) error {
	if err := ValidateSchemaName(name); err != nil {
		return err
	}

	t.SchemaName = name
	return nil
}

// IsActive checks if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsSuspended checks if the tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// IsTrial checks if the tenant is in trial period
func (t *Tenant) IsTrial() bool {
	return t.Status == TenantStatusTrial
}

// IsTrialExpired checks if trial period has expired
func (t *Tenant) IsTrialExpired() bool {
	if t.TrialEndsAt == nil {
		return false
	}
	return time.Now().After(*t.TrialEndsAt)
}

// HasCustomDomain checks if tenant has a custom domain
func (t *Tenant) HasCustomDomain() bool {
	return t.CustomDomain != ""
}

// IsCustomDomainVerified checks if custom domain is verified
func (t *Tenant) IsCustomDomainVerified() bool {
	return t.HasCustomDomain() && t.CustomDomainVerified
}

// Suspend suspends the tenant account
func (t *Tenant) Suspend(reason string) error {
	if t.IsSuspended() {
		return ErrTenantAlreadySuspended
	}

	t.Status = TenantStatusSuspended
	t.UpdatedAt = time.Now()

	if t.Metadata == nil {
		t.Metadata = make(map[string]any)
	}
	t.Metadata["suspension_reason"] = reason
	t.Metadata["suspended_at"] = time.Now()

	return nil
}

// Activate activates the tenant account
func (t *Tenant) Activate() error {
	if t.IsActive() {
		return ErrTenantAlreadyActive
	}

	t.Status = TenantStatusActive
	t.UpdatedAt = time.Now()

	if t.Metadata != nil {
		delete(t.Metadata, "suspension_reason")
		delete(t.Metadata, "suspended_at")
	}

	return nil
}

// UpgradePlan upgrades the tenant subscription plan
func (t *Tenant) UpgradePlan(newPlan SubscriptionPlan) error {
	if !newPlan.IsValid() {
		return ErrInvalidSubscriptionPlan
	}

	if t.Plan == newPlan {
		return ErrSameSubscriptionPlan
	}

	t.Plan = newPlan
	t.UpdatedAt = time.Now()

	// Remove trial status if upgrading from trial
	if t.IsTrial() {
		t.Status = TenantStatusActive
		t.TrialEndsAt = nil
	}

	return nil
}

// SetCustomDomain sets a custom domain for the tenant
func (t *Tenant) SetCustomDomain(domain string) error {
	if domain == "" {
		return ErrInvalidCustomDomain
	}

	t.CustomDomain = domain
	t.CustomDomainVerified = false
	t.CustomDomainVerifiedAt = nil
	t.UpdatedAt = time.Now()

	return nil
}

// VerifyCustomDomain marks the custom domain as verified
func (t *Tenant) VerifyCustomDomain() error {
	if !t.HasCustomDomain() {
		return ErrCustomDomainNotSet
	}

	if t.IsCustomDomainVerified() {
		return ErrCustomDomainAlreadyVerified
	}

	now := time.Now()
	t.CustomDomainVerified = true
	t.CustomDomainVerifiedAt = &now
	t.UpdatedAt = now

	return nil
}

// RemoveCustomDomain removes the custom domain
func (t *Tenant) RemoveCustomDomain() {
	t.CustomDomain = ""
	t.CustomDomainVerified = false
	t.CustomDomainVerifiedAt = nil
	t.UpdatedAt = time.Now()
}

// UpdateSettings updates tenant settings
func (t *Tenant) UpdateSettings(key string, value any) {
	if t.Settings == nil {
		t.Settings = make(map[string]any)
	}
	t.Settings[key] = value
	t.UpdatedAt = time.Now()
}

// GetSetting retrieves a tenant setting
func (t *Tenant) GetSetting(key string) (any, bool) {
	if t.Settings == nil {
		return nil, false
	}
	value, exists := t.Settings[key]
	return value, exists
}

// IsValid checks if tenant status is valid
func (s TenantStatus) IsValid() bool {
	switch s {
	case TenantStatusPending, TenantStatusProvisioning, TenantStatusFailed,
		TenantStatusActive, TenantStatusSuspended, TenantStatusInactive, TenantStatusTrial:
		return true
	default:
		return false
	}
}

// IsValid checks if subscription plan is valid
func (p SubscriptionPlan) IsValid() bool {
	switch p {
	case PlanFree, PlanStarter, PlanPro, PlanEnterprise:
		return true
	default:
		return false
	}
}

// GetFeatureLimits returns feature limits for the subscription plan
func (p SubscriptionPlan) GetFeatureLimits() FeatureLimits {
	switch p {
	case PlanFree:
		return FeatureLimits{
			MaxUsers:     1,
			MaxWebsites:  1,
			MaxStorage:   100 * 1024 * 1024, // 100MB
			CustomDomain: false,
			APIAccess:    false,
		}
	case PlanStarter:
		return FeatureLimits{
			MaxUsers:     5,
			MaxWebsites:  3,
			MaxStorage:   1 * 1024 * 1024 * 1024, // 1GB
			CustomDomain: false,
			APIAccess:    true,
		}
	case PlanPro:
		return FeatureLimits{
			MaxUsers:     20,
			MaxWebsites:  10,
			MaxStorage:   10 * 1024 * 1024 * 1024, // 10GB
			CustomDomain: true,
			APIAccess:    true,
		}
	case PlanEnterprise:
		return FeatureLimits{
			MaxUsers:     -1, // Unlimited
			MaxWebsites:  -1, // Unlimited
			MaxStorage:   -1, // Unlimited
			CustomDomain: true,
			APIAccess:    true,
		}
	default:
		return FeatureLimits{}
	}
}

// FeatureLimits represents feature limits for a subscription plan
type FeatureLimits struct {
	MaxUsers     int   `json:"max_users"`    // -1 for unlimited
	MaxWebsites  int   `json:"max_websites"` // -1 for unlimited
	MaxStorage   int64 `json:"max_storage"`  // in bytes, -1 for unlimited
	CustomDomain bool  `json:"custom_domain"`
	APIAccess    bool  `json:"api_access"`
}

// HasFeature checks if a feature is available in the limits
func (f *FeatureLimits) HasFeature(feature string) bool {
	switch feature {
	case "custom_domain":
		return f.CustomDomain
	case "api_access":
		return f.APIAccess
	default:
		return false
	}
}

// CanAddUser checks if tenant can add more users
func (f *FeatureLimits) CanAddUser(currentCount int) bool {
	if f.MaxUsers == -1 {
		return true // Unlimited
	}
	return currentCount < f.MaxUsers
}

// CanAddWebsite checks if tenant can add more websites
func (f *FeatureLimits) CanAddWebsite(currentCount int) bool {
	if f.MaxWebsites == -1 {
		return true // Unlimited
	}
	return currentCount < f.MaxWebsites
}

// HasStorageAvailable checks if tenant has storage available
func (f *FeatureLimits) HasStorageAvailable(currentUsage, additionalNeeded int64) bool {
	if f.MaxStorage == -1 {
		return true // Unlimited
	}
	return (currentUsage + additionalNeeded) <= f.MaxStorage
}
