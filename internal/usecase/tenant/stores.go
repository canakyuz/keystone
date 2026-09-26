package tenant

import (
	"context"

	"github.com/canakyuz/keystone/internal/domain/tenant"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
)

// Store is what the tenant service needs from the tenant repository. It is declared here,
// by its consumer, and names only the methods the service calls.
type Store interface {
	CountByPlan(ctx context.Context, plan tenant.SubscriptionPlan) (int64, error)
	CountByStatus(ctx context.Context, status tenant.TenantStatus) (int64, error)
	DeleteAudited(ctx context.Context, id string, entry auditrepo.Entry) error
	ExistsByCustomDomain(ctx context.Context, domain string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	GetByID(ctx context.Context, id string) (*tenant.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error)
	List(ctx context.Context, filters tenantRepo.ListFilters) ([]*tenant.Tenant, int64, error)
	UpdateAudited(ctx context.Context, t *tenant.Tenant, entry auditrepo.Entry) error
}

// TemplateSource supplies the SQL applied to a new tenant's schema for its plan.
type TemplateSource interface {
	GetTemplateByPlan(ctx context.Context, plan string) (string, error)
}
