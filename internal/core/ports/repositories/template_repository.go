package repositories

import (
	"context"
	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/template"
)

// TemplateRepository defines the interface for template persistence operations.
type TemplateRepository interface {
	Create(ctx context.Context, tmpl *template.Template) error
	GetByID(ctx context.Context, id shared.TemplateID) (*template.Template, error)
	GetByNameAndTenant(ctx context.Context, name string, tenantID shared.TenantID) (*template.Template, error)
	Update(ctx context.Context, tmpl *template.Template) error
	Delete(ctx context.Context, id shared.TemplateID, tenantID shared.TenantID) error
	Search(ctx context.Context, criteria TemplateSearchCriteria) ([]*template.Template, int, error)
	ListByTenant(ctx context.Context, tenantID shared.TenantID, criteria TemplateListCriteria) ([]*template.Template, int, error)

	CreateInstallation(ctx context.Context, installation *template.Installation) error
	GetInstallationByTemplateAndTenant(ctx context.Context, templateID shared.TemplateID, tenantID shared.TenantID) (*template.Installation, error)
}

// TemplateSearchCriteria defines the parameters for searching public templates.
type TemplateSearchCriteria struct {
	TenantID     shared.TenantID
	Query        string
	Category     *template.Category
	Visibility   *template.Visibility
	PricingModel *template.PricingModel
	Tags         []string
	Page         int
	PageSize     int
}

// TemplateListCriteria defines the parameters for listing templates within a tenant.
type TemplateListCriteria struct {
	Status   *template.Status
	Category *template.Category
	Page     int
	PageSize int
}
