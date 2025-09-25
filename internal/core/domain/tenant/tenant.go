package tenant

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	. "nexspaces-api/internal/core/domain/shared"
)

// Tenant represents a tenant in the multi-tenant system
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusSuspended Status = "suspended"
	StatusCancelled Status = "cancelled"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID             TenantID               `json:"id" db:"id"`
	Name           string                 `json:"name" db:"name"`
	Slug           TenantSlug             `json:"slug" db:"slug"`
	CustomDomain   *Domain                `json:"custom_domain,omitempty" db:"custom_domain"`
	Status         Status                 `json:"status" db:"status"`
	SubscriptionID *SubscriptionID        `json:"subscription_id,omitempty" db:"subscription_id"`
	Settings       map[string]interface{} `json:"settings" db:"settings"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      *Timestamp             `json:"created_at" db:"created_at"`
	UpdatedAt      *Timestamp             `json:"updated_at" db:"updated_at"`
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
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	CustomDomain string                 `json:"custom_domain,omitempty"`
	Status       Status                 `json:"status"`
	Settings     map[string]interface{} `json:"settings"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ToResponse converts a Tenant domain object to a TenantResponse
func (t *Tenant) ToResponse() *TenantResponse {
	customDomain := ""
	if t.CustomDomain != nil {
		customDomain = t.CustomDomain.String()
	}

	return &TenantResponse{
		ID:           t.ID.String(),
		Name:         t.Name,
		Slug:         t.Slug.String(),
		CustomDomain: customDomain,
		Status:       t.Status,
		Settings:     t.Settings,
		CreatedAt:    t.CreatedAt.Time(),
		UpdatedAt:    t.UpdatedAt.Time(),
	}
}

// GetDomain returns the domain for this tenant (custom domain or slug-based)
func (t *Tenant) GetDomain() string {
	if t.CustomDomain != nil {
		return t.CustomDomain.String()
	}
	return t.Slug.String() + ".nexpaces.com"
}

// HasFeature checks if a feature is enabled for this tenant
func (t *Tenant) HasFeature(feature string) bool {
	if t.Settings == nil {
		return false
	}
	if features, ok := t.Settings["features"].(map[string]interface{}); ok {
		if enabled, exists := features[feature].(bool); exists {
			return enabled
		}
	}
	return false
}

// NewTenant creates a new Tenant instance
func NewTenant(req CreateTenantRequest) *Tenant {
	slug, _ := NewTenantSlug(req.Slug)

	settings := map[string]interface{}{
		"theme":    "default",
		"language": "en",
		"timezone": "UTC",
		"features": map[string]interface{}{
			"templates": true,
			"users":     true,
			"analytics": true,
		},
		"custom_config": map[string]interface{}{},
	}

	tenant := &Tenant{
		ID:        NewTenantID(uuid.New()),
		Name:      req.Name,
		Slug:      *slug,
		Status:    StatusActive,
		Settings:  settings,
		Metadata:  make(map[string]interface{}),
		CreatedAt: NewTimestamp(),
		UpdatedAt: NewTimestamp(),
	}

	if req.CustomDomain != nil {
		domain, _ := NewDomain(*req.CustomDomain)
		tenant.CustomDomain = domain
	}

	return tenant
}

// ParseStatus parses a string into a Status
func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusPending, StatusActive, StatusInactive, StatusSuspended, StatusCancelled:
		return Status(s), nil
	default:
		return "", fmt.Errorf("invalid tenant status: %s", s)
	}
}
