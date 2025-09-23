package repositories

import (
	"context"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
)

// TenantRepository defines the interface for tenant data access
type TenantRepository interface {
	// Create creates a new tenant
	Create(ctx context.Context, tenant *tenant.Tenant) error

	// GetByID retrieves a tenant by ID
	GetByID(ctx context.Context, id tenant.TenantID) (*tenant.Tenant, error)

	// GetBySlug retrieves a tenant by slug
	GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error)

	// Update updates an existing tenant
	Update(ctx context.Context, tenant *tenant.Tenant) error

	// Delete deletes a tenant (soft delete)
	Delete(ctx context.Context, id tenant.TenantID) error

	// List retrieves tenants with filtering and pagination
	List(ctx context.Context, filter TenantFilter) (*TenantList, error)

	// ExistsBySlug checks if a tenant with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// Count returns the total number of tenants matching the filter
	Count(ctx context.Context, filter TenantFilter) (int64, error)

	// GetActiveCount returns the number of active tenants
	GetActiveCount(ctx context.Context) (int64, error)

	// GetByCustomDomain retrieves tenant by custom domain
	GetByCustomDomain(ctx context.Context, domain string) (*tenant.Tenant, error)

	// GetTenantsForUser retrieves tenants that user has access to
	GetTenantsForUser(ctx context.Context, userID shared.ID) ([]*tenant.Tenant, error)

	// UpdateLastActivity updates tenant's last activity timestamp
	UpdateLastActivity(ctx context.Context, id tenant.TenantID) error

	// GetMetrics retrieves tenant metrics
	GetMetrics(ctx context.Context, id tenant.TenantID) (*TenantMetrics, error)
}

// TenantFilter represents filtering options for tenant queries
type TenantFilter struct {
	// Status filtering
	Status *shared.Status

	// Search term (searches in name and slug)
	Search string

	// Created date range
	CreatedAfter  *shared.Timestamp
	CreatedBefore *shared.Timestamp

	// Plan filtering
	PlanID *string

	// Custom domain filtering
	HasCustomDomain *bool

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // name, created_at, updated_at
	SortOrder string // asc, desc

	// Include related data
	IncludeMetrics bool
	IncludeUsers   bool
}

// TenantList represents a paginated list of tenants
type TenantList struct {
	Items      []*tenant.Tenant `json:"items"`
	Total      int64            `json:"total"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
	HasMore    bool             `json:"has_more"`
	TotalPages int              `json:"total_pages"`
}

// TenantMetrics represents tenant usage metrics
type TenantMetrics struct {
	TenantID       tenant.TenantID   `json:"tenant_id"`
	UserCount      int               `json:"user_count"`
	TemplateCount  int               `json:"template_count"`
	StorageUsed    int64             `json:"storage_used"`   // bytes
	BandwidthUsed  int64             `json:"bandwidth_used"` // bytes (current period)
	APICallsUsed   int64             `json:"api_calls_used"` // current period
	LastActivityAt *shared.Timestamp `json:"last_activity_at"`
	CreatedAt      shared.Timestamp  `json:"created_at"`
	UpdatedAt      shared.Timestamp  `json:"updated_at"`
}

// Validation methods for filter
func (f *TenantFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 1000 {
		return shared.NewValidationError("limit must be between 0 and 1000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	if f.SortBy != "" {
		validSortFields := []string{"name", "created_at", "updated_at", "status"}
		valid := false
		for _, field := range validSortFields {
			if f.SortBy == field {
				valid = true
				break
			}
		}
		if !valid {
			return shared.NewValidationError("invalid sort field")
		}
	}

	if f.SortOrder != "" && f.SortOrder != "asc" && f.SortOrder != "desc" {
		return shared.NewValidationError("sort order must be 'asc' or 'desc'")
	}

	return nil
}

// Apply default values
func (f *TenantFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	if f.SortBy == "" {
		f.SortBy = "created_at"
	}

	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}
