package repositories

import (
	"context"
	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/tenant"
)

// TenantRepository defines the interface for tenant data operations
type TenantRepository interface {
	// Create creates a new tenant
	Create(ctx context.Context, tenant *tenant.Tenant) error

	// GetByID retrieves a tenant by ID
	GetByID(ctx context.Context, tenantID uuid.UUID) (*tenant.Tenant, error)

	// GetBySlug retrieves a tenant by slug
	GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error)

	// GetByCustomDomain retrieves a tenant by custom domain
	GetByCustomDomain(ctx context.Context, domain string) (*tenant.Tenant, error)

	// Update updates an existing tenant
	Update(ctx context.Context, tenant *tenant.Tenant) error

	// Delete soft deletes a tenant
	Delete(ctx context.Context, tenantID uuid.UUID) error

	// List retrieves all tenants with pagination
	List(ctx context.Context, criteria TenantListCriteria) ([]*tenant.Tenant, int, error)

	// Simple List for basic pagination (backward compatibility)
	ListSimple(ctx context.Context, limit, offset int) ([]*tenant.Tenant, error)

	// Count returns the total number of tenants
	Count(ctx context.Context) (int, error)

	// GetActiveTenantsCount returns the number of active tenants
	GetActiveTenantsCount(ctx context.Context) (int, error)

	// ActivateTenant activates a tenant
	ActivateTenant(ctx context.Context, tenantID uuid.UUID) error

	// DeactivateTenant deactivates a tenant
	DeactivateTenant(ctx context.Context, tenantID uuid.UUID) error

	// UpdateSettings updates tenant settings
	UpdateSettings(ctx context.Context, tenantID uuid.UUID, settings tenant.Settings) error

	// IsSlugAvailable checks if a slug is available
	IsSlugAvailable(ctx context.Context, slug string) (bool, error)

	// IsDomainAvailable checks if a custom domain is available
	IsDomainAvailable(ctx context.Context, domain string) (bool, error)

	// Search searches tenants by name or slug
	Search(ctx context.Context, query string, limit, offset int) ([]*tenant.Tenant, error)

	// GetTenantsByCreatedDate retrieves tenants created within a date range
	GetTenantsByCreatedDate(ctx context.Context, startDate, endDate string) ([]*tenant.Tenant, error)
}

// TenantListCriteria represents the criteria for listing tenants
type TenantListCriteria struct {
	Page     int
	PageSize int
	Status   *tenant.Status
	Search   string
}
