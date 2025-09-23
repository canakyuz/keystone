package tenant

import (
	"github.com/google/uuid"
	"time"
)

// Tenant represents a tenant in the multi-tenant system
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Slug           string     `json:"slug" db:"slug"`
	CustomDomain   string     `json:"custom_domain,omitempty" db:"custom_domain"`
	Status         Status     `json:"status" db:"status"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty" db:"subscription_id"`
	Settings       Settings   `json:"settings" db:"settings"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Settings represents tenant-specific configuration
type Settings struct {
	Theme        string          `json:"theme"`
	Language     string          `json:"language"`
	Timezone     string          `json:"timezone"`
	Features     map[string]bool `json:"features"`
	CustomConfig map[string]any  `json:"custom_config"`
}

// CreateTenantRequest represents the data needed to create a new tenant
type CreateTenantRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=100"`
	Slug         string  `json:"slug" validate:"required,min=2,max=50,alphanum"`
	CustomDomain *string `json:"custom_domain,omitempty" validate:"omitempty,url"`
}

// UpdateTenantRequest represents the data that can be updated for a tenant
type UpdateTenantRequest struct {
	Name         *string   `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	CustomDomain *string   `json:"custom_domain,omitempty" validate:"omitempty,url"`
	Status       *Status   `json:"status,omitempty"`
	Settings     *Settings `json:"settings,omitempty"`
}

// TenantResponse represents the tenant data returned in API responses
type TenantResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	CustomDomain string    `json:"custom_domain,omitempty"`
	Status       Status    `json:"status"`
	Settings     Settings  `json:"settings"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToResponse converts a Tenant domain object to a TenantResponse
func (t *Tenant) ToResponse() *TenantResponse {
	return &TenantResponse{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		CustomDomain: t.CustomDomain,
		Status:       t.Status,
		Settings:     t.Settings,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// GetDomain returns the domain for this tenant (custom domain or slug-based)
func (t *Tenant) GetDomain() string {
	if t.CustomDomain != "" {
		return t.CustomDomain
	}
	return t.Slug + ".nexpaces.com"
}

// HasFeature checks if a feature is enabled for this tenant
func (t *Tenant) HasFeature(feature string) bool {
	if t.Settings.Features == nil {
		return false
	}
	return t.Settings.Features[feature]
}

// NewTenant creates a new Tenant instance
func NewTenant(req CreateTenantRequest) *Tenant {
	settings := Settings{
		Theme:        "default",
		Language:     "en",
		Timezone:     "UTC",
		Features:     make(map[string]bool),
		CustomConfig: make(map[string]any),
	}

	// Enable default features
	settings.Features["templates"] = true
	settings.Features["users"] = true
	settings.Features["analytics"] = true

	tenant := &Tenant{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		Status:    StatusActive,
		Settings:  settings,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.CustomDomain != nil {
		tenant.CustomDomain = *req.CustomDomain
	}

	return tenant
}
