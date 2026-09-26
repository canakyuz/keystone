package registry

import (
	"context"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// The stores the registry usecases need. They are declared here, by their consumer, and
// name only the methods the usecases call; the repository package returns concrete types.

// ModuleRepository reads the module catalogue.
type ModuleRepository interface {
	GetByID(ctx context.Context, id string) (*registry.Module, error)
	GetBySlug(ctx context.Context, slug string) (*registry.Module, error)
	List(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error)
	ListPublic(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error)
	Search(ctx context.Context, query string, filters registry.ModuleFilters) ([]*registry.Module, error)
}

// ToolRepository reads the tool catalogue.
type ToolRepository interface {
	GetByID(ctx context.Context, id string) (*registry.Tool, error)
	GetBySlug(ctx context.Context, slug string) (*registry.Tool, error)
	List(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error)
	ListPublic(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error)
	Search(ctx context.Context, query string, filters registry.ToolFilters) ([]*registry.Tool, error)
}

// TenantModuleRepository records a tenant's module installations.
type TenantModuleRepository interface {
	Activate(ctx context.Context, tenantID, moduleID, activatedBy string) error
	CompleteSetup(ctx context.Context, tenantID, moduleID string) error
	Create(ctx context.Context, tenantModule *registry.TenantModule) error
	Deactivate(ctx context.Context, tenantID, moduleID, deactivatedBy string) error
	GetByTenantAndModule(ctx context.Context, tenantID, moduleID string) (*registry.TenantModule, error)
	IsModuleActivated(ctx context.Context, tenantID, moduleID string) (bool, error)
	ListActivatedByTenant(ctx context.Context, tenantID string) ([]*registry.TenantModule, error)
	Uninstall(ctx context.Context, tenantID, moduleID string) error
}

// TenantToolRepository records a tenant's tool installations.
type TenantToolRepository interface {
	Activate(ctx context.Context, tenantID, toolID, activatedBy string) error
	CompleteSetup(ctx context.Context, tenantID, toolID string) error
	Create(ctx context.Context, tenantTool *registry.TenantTool) error
	Deactivate(ctx context.Context, tenantID, toolID, deactivatedBy string) error
	GetByTenantAndTool(ctx context.Context, tenantID, toolID string) (*registry.TenantTool, error)
	IsToolActivated(ctx context.Context, tenantID, toolID string) (bool, error)
	ListActivatedByTenant(ctx context.Context, tenantID string) ([]*registry.TenantTool, error)
	Uninstall(ctx context.Context, tenantID, toolID string) error
}
