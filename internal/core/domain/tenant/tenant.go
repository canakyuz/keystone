package tenant

import (
	"fmt"
	"strings"
	"time"

	"nexspaces-api/internal/core/domain/shared"
)

// TenantID represents a unique tenant identifier
type TenantID = shared.ID

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	id           TenantID
	name         string
	slug         shared.Slug
	customDomain *string
	planID       *string
	status       shared.Status
	settings     shared.Settings
	metadata     map[string]interface{}
	createdAt    shared.Timestamp
	updatedAt    shared.Timestamp
	version      int64 // For optimistic locking
}

// TenantParams contains parameters for creating a new tenant
type TenantParams struct {
	Name         string
	Slug         string
	CustomDomain *string
	PlanID       *string
	Settings     shared.Settings
	Metadata     map[string]interface{}
}

// NewTenant creates a new tenant entity
func NewTenant(params TenantParams) (*Tenant, error) {
	// Validate required fields
	if strings.TrimSpace(params.Name) == "" {
		return nil, shared.NewValidationError("tenant name is required")
	}

	if len(params.Name) > 100 {
		return nil, shared.NewValidationError("tenant name must be 100 characters or less")
	}

	// Create slug
	slug, err := shared.NewSlug(params.Slug)
	if err != nil {
		return nil, shared.WrapDomainError(err, shared.ValidationError, "invalid tenant slug")
	}

	// Validate custom domain if provided
	if params.CustomDomain != nil && *params.CustomDomain != "" {
		if err := validateDomain(*params.CustomDomain); err != nil {
			return nil, shared.WrapDomainError(err, shared.ValidationError, "invalid custom domain")
		}
	}

	// Initialize settings if nil
	if params.Settings == nil {
		params.Settings = make(shared.Settings)
	}

	// Initialize metadata if nil
	if params.Metadata == nil {
		params.Metadata = make(map[string]interface{})
	}

	now := shared.Now()

	return &Tenant{
		id:           shared.NewID(),
		name:         strings.TrimSpace(params.Name),
		slug:         slug,
		customDomain: params.CustomDomain,
		planID:       params.PlanID,
		status:       shared.StatusActive,
		settings:     params.Settings,
		metadata:     params.Metadata,
		createdAt:    now,
		updatedAt:    now,
		version:      1,
	}, nil
}

// ReconstituteTenant recreates a tenant from stored data (for repository pattern)
func ReconstituteTenant(
	id TenantID,
	name string,
	slug shared.Slug,
	customDomain *string,
	planID *string,
	status shared.Status,
	settings shared.Settings,
	metadata map[string]interface{},
	createdAt shared.Timestamp,
	updatedAt shared.Timestamp,
	version int64,
) *Tenant {
	return &Tenant{
		id:           id,
		name:         name,
		slug:         slug,
		customDomain: customDomain,
		planID:       planID,
		status:       status,
		settings:     settings,
		metadata:     metadata,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		version:      version,
	}
}

// Getters
func (t *Tenant) ID() TenantID {
	return t.id
}

func (t *Tenant) Name() string {
	return t.name
}

func (t *Tenant) Slug() shared.Slug {
	return t.slug
}

func (t *Tenant) CustomDomain() *string {
	return t.customDomain
}

func (t *Tenant) PlanID() *string {
	return t.planID
}

func (t *Tenant) Status() shared.Status {
	return t.status
}

func (t *Tenant) Settings() shared.Settings {
	return t.settings
}

func (t *Tenant) Metadata() map[string]interface{} {
	return t.metadata
}

func (t *Tenant) CreatedAt() shared.Timestamp {
	return t.createdAt
}

func (t *Tenant) UpdatedAt() shared.Timestamp {
	return t.updatedAt
}

func (t *Tenant) Version() int64 {
	return t.version
}

// Business Methods

// UpdateName updates the tenant name
func (t *Tenant) UpdateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return shared.NewValidationError("tenant name cannot be empty")
	}

	if len(name) > 100 {
		return shared.NewValidationError("tenant name must be 100 characters or less")
	}

	t.name = strings.TrimSpace(name)
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// UpdateSlug updates the tenant slug
func (t *Tenant) UpdateSlug(slug string) error {
	newSlug, err := shared.NewSlug(slug)
	if err != nil {
		return shared.WrapDomainError(err, shared.ValidationError, "invalid tenant slug")
	}

	t.slug = newSlug
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// SetCustomDomain sets or updates the custom domain
func (t *Tenant) SetCustomDomain(domain *string) error {
	if domain != nil && *domain != "" {
		if err := validateDomain(*domain); err != nil {
			return shared.WrapDomainError(err, shared.ValidationError, "invalid custom domain")
		}
	}

	t.customDomain = domain
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// UpdateSettings updates tenant settings with merge strategy
func (t *Tenant) UpdateSettings(newSettings shared.Settings) error {
	if !t.IsActive() {
		return shared.NewBusinessRuleError("cannot update settings for inactive tenant")
	}

	// Validate settings
	if err := newSettings.Validate(); err != nil {
		return shared.WrapDomainError(err, shared.ValidationError, "invalid settings")
	}

	// Merge with existing settings
	t.settings = t.settings.Merge(newSettings)
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// SetPlan updates the tenant's subscription plan
func (t *Tenant) SetPlan(planID string) error {
	if strings.TrimSpace(planID) == "" {
		return shared.NewValidationError("plan ID cannot be empty")
	}

	t.planID = &planID
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// Activate activates the tenant
func (t *Tenant) Activate() error {
	if t.status == shared.StatusDeleted {
		return shared.NewBusinessRuleError("cannot activate deleted tenant")
	}

	if t.status == shared.StatusActive {
		return nil // Already active
	}

	t.status = shared.StatusActive
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// Suspend suspends the tenant
func (t *Tenant) Suspend() error {
	if t.status == shared.StatusDeleted {
		return shared.NewBusinessRuleError("cannot suspend deleted tenant")
	}

	if t.status == shared.StatusSuspended {
		return nil // Already suspended
	}

	t.status = shared.StatusSuspended
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// Deactivate deactivates the tenant
func (t *Tenant) Deactivate() error {
	if t.status == shared.StatusDeleted {
		return shared.NewBusinessRuleError("cannot deactivate deleted tenant")
	}

	t.status = shared.StatusInactive
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// Delete marks the tenant as deleted (soft delete)
func (t *Tenant) Delete() error {
	if t.status == shared.StatusDeleted {
		return nil // Already deleted
	}

	t.status = shared.StatusDeleted
	t.updatedAt = shared.Now()
	t.version++

	return nil
}

// SetMetadata sets metadata key-value pair
func (t *Tenant) SetMetadata(key string, value interface{}) {
	if t.metadata == nil {
		t.metadata = make(map[string]interface{})
	}

	t.metadata[key] = value
	t.updatedAt = shared.Now()
	t.version++
}

// GetMetadata retrieves metadata value
func (t *Tenant) GetMetadata(key string) (interface{}, bool) {
	value, exists := t.metadata[key]
	return value, exists
}

// Business Logic Queries

// IsActive checks if tenant is active
func (t *Tenant) IsActive() bool {
	return t.status == shared.StatusActive
}

// IsSuspended checks if tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.status == shared.StatusSuspended
}

// IsDeleted checks if tenant is deleted
func (t *Tenant) IsDeleted() bool {
	return t.status == shared.StatusDeleted
}

// CanAccessFeature checks if tenant can access a specific feature
func (t *Tenant) CanAccessFeature(feature string) bool {
	if !t.IsActive() {
		return false
	}

	// Check feature access based on plan or settings
	if featureAccess, exists := t.settings.Get("features"); exists {
		if features, ok := featureAccess.(map[string]interface{}); ok {
			if access, exists := features[feature]; exists {
				if enabled, ok := access.(bool); ok {
					return enabled
				}
			}
		}
	}

	// Default behavior - check if it's a basic feature
	basicFeatures := []string{"users", "basic_templates", "dashboard"}
	for _, basicFeature := range basicFeatures {
		if feature == basicFeature {
			return true
		}
	}

	return false
}

// GetFeatureLimit returns the limit for a specific feature
func (t *Tenant) GetFeatureLimit(feature string) int {
	if !t.IsActive() {
		return 0
	}

	if limits, exists := t.settings.Get("limits"); exists {
		if limitMap, ok := limits.(map[string]interface{}); ok {
			if limit, exists := limitMap[feature]; exists {
				if limitInt, ok := limit.(float64); ok {
					return int(limitInt)
				}
				if limitInt, ok := limit.(int); ok {
					return limitInt
				}
			}
		}
	}

	// Default limits
	defaultLimits := map[string]int{
		"users":      10,
		"templates":  5,
		"storage_mb": 1000,
		"api_calls":  10000,
	}

	if limit, exists := defaultLimits[feature]; exists {
		return limit
	}

	return 0
}

// HasCustomDomain checks if tenant has a custom domain
func (t *Tenant) HasCustomDomain() bool {
	return t.customDomain != nil && *t.customDomain != ""
}

// GetPrimaryDomain returns the primary domain (custom domain or slug-based)
func (t *Tenant) GetPrimaryDomain() string {
	if t.HasCustomDomain() {
		return *t.customDomain
	}
	return fmt.Sprintf("%s.nexpaces.com", t.slug.String())
}

// Utility functions

// validateDomain validates domain format
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Basic domain validation
	if len(domain) > 253 {
		return fmt.Errorf("domain too long")
	}

	if strings.Contains(domain, " ") {
		return fmt.Errorf("domain cannot contain spaces")
	}

	// Add more sophisticated domain validation as needed
	return nil
}

// Events (for event sourcing/domain events)

// TenantCreatedEvent represents a tenant creation event
type TenantCreatedEvent struct {
	TenantID  TenantID
	Name      string
	Slug      string
	CreatedAt time.Time
}

// TenantUpdatedEvent represents a tenant update event
type TenantUpdatedEvent struct {
	TenantID  TenantID
	Name      string
	UpdatedAt time.Time
	Version   int64
}

// TenantStatusChangedEvent represents a tenant status change event
type TenantStatusChangedEvent struct {
	TenantID  TenantID
	OldStatus shared.Status
	NewStatus shared.Status
	ChangedAt time.Time
}

// GenerateCreatedEvent generates a tenant created event
func (t *Tenant) GenerateCreatedEvent() TenantCreatedEvent {
	return TenantCreatedEvent{
		TenantID:  t.id,
		Name:      t.name,
		Slug:      t.slug.String(),
		CreatedAt: t.createdAt.Time(),
	}
}

// GenerateUpdatedEvent generates a tenant updated event
func (t *Tenant) GenerateUpdatedEvent() TenantUpdatedEvent {
	return TenantUpdatedEvent{
		TenantID:  t.id,
		Name:      t.name,
		UpdatedAt: t.updatedAt.Time(),
		Version:   t.version,
	}
}

// GenerateStatusChangedEvent generates a status changed event
func (t *Tenant) GenerateStatusChangedEvent(oldStatus shared.Status) TenantStatusChangedEvent {
	return TenantStatusChangedEvent{
		TenantID:  t.id,
		OldStatus: oldStatus,
		NewStatus: t.status,
		ChangedAt: t.updatedAt.Time(),
	}
}
