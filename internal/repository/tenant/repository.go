package tenant

import (
	"github.com/canakyuz/keystone/internal/domain/tenant"
)

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
