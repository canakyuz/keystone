package tenant

import (
	"context"

	"nexspaces-api/internal/domain/tenant"
)

// Repository defines the interface for tenant data operations
type Repository interface {
	// Create creates a new tenant
	Create(ctx context.Context, t *tenant.Tenant) error

	// GetByID retrieves a tenant by ID
	GetByID(ctx context.Context, id string) (*tenant.Tenant, error)

	// GetBySlug retrieves a tenant by slug
	GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error)

	// GetByEmail retrieves a tenant by email
	GetByEmail(ctx context.Context, email string) (*tenant.Tenant, error)

	// GetByCustomDomain retrieves a tenant by custom domain
	GetByCustomDomain(ctx context.Context, domain string) (*tenant.Tenant, error)

	// List retrieves all tenants with pagination
	List(ctx context.Context, filters ListFilters) ([]*tenant.Tenant, int64, error)

	// Update updates an existing tenant
	Update(ctx context.Context, t *tenant.Tenant) error

	// Delete soft deletes a tenant
	Delete(ctx context.Context, id string) error

	// ExistsBySlug checks if a tenant with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// ExistsByEmail checks if a tenant with the given email exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByCustomDomain checks if a tenant with the given custom domain exists
	ExistsByCustomDomain(ctx context.Context, domain string) (bool, error)

	// CountByStatus counts tenants by status
	CountByStatus(ctx context.Context, status tenant.TenantStatus) (int64, error)

	// CountByPlan counts tenants by subscription plan
	CountByPlan(ctx context.Context, plan tenant.SubscriptionPlan) (int64, error)
}

// ListFilters represents filters for listing tenants
type ListFilters struct {
	// Pagination
	Limit  int
	Offset int

	// Filters
	Status *tenant.TenantStatus
	Plan   *tenant.SubscriptionPlan
	Search string // Search in name, slug, email

	// Sorting
	SortBy    string // created_at, name, slug, email
	SortOrder string // asc, desc
}
