package registry

import "context"

// ModuleRepository defines module catalog operations
type ModuleRepository interface {
	// CRUD operations
	Create(ctx context.Context, module *Module) error
	GetByID(ctx context.Context, id string) (*Module, error)
	GetByCode(ctx context.Context, code string) (*Module, error)
	GetBySlug(ctx context.Context, slug string) (*Module, error)
	Update(ctx context.Context, module *Module) error
	Delete(ctx context.Context, id string) error

	// Catalog queries
	List(ctx context.Context, filters ModuleFilters) ([]*Module, error)
	ListPublic(ctx context.Context, filters ModuleFilters) ([]*Module, error)
	Search(ctx context.Context, query string, filters ModuleFilters) ([]*Module, error)

	// Statistics
	GetInstallCount(ctx context.Context, moduleID string) (int, error)
	UpdateInstallCount(ctx context.Context, moduleID string, delta int) error
}

// ToolRepository defines tool catalog operations
type ToolRepository interface {
	// CRUD operations
	Create(ctx context.Context, tool *Tool) error
	GetByID(ctx context.Context, id string) (*Tool, error)
	GetByCode(ctx context.Context, code string) (*Tool, error)
	GetBySlug(ctx context.Context, slug string) (*Tool, error)
	Update(ctx context.Context, tool *Tool) error
	Delete(ctx context.Context, id string) error

	// Catalog queries
	List(ctx context.Context, filters ToolFilters) ([]*Tool, error)
	ListPublic(ctx context.Context, filters ToolFilters) ([]*Tool, error)
	Search(ctx context.Context, query string, filters ToolFilters) ([]*Tool, error)

	// Statistics
	GetInstallCount(ctx context.Context, toolID string) (int, error)
	UpdateInstallCount(ctx context.Context, toolID string, delta int) error
}

// TenantModuleRepository defines tenant module activation operations
type TenantModuleRepository interface {
	// CRUD operations
	Create(ctx context.Context, tenantModule *TenantModule) error
	GetByID(ctx context.Context, id string) (*TenantModule, error)
	GetByTenantAndModule(ctx context.Context, tenantID, moduleID string) (*TenantModule, error)
	Update(ctx context.Context, tenantModule *TenantModule) error
	Delete(ctx context.Context, id string) error

	// Tenant-scoped queries
	ListByTenant(ctx context.Context, tenantID string, filters TenantModuleFilters) ([]*TenantModule, error)
	ListActivatedByTenant(ctx context.Context, tenantID string) ([]*TenantModule, error)
	IsModuleActivated(ctx context.Context, tenantID, moduleID string) (bool, error)

	// Activation operations
	Activate(ctx context.Context, tenantID, moduleID, activatedBy string) error
	Deactivate(ctx context.Context, tenantID, moduleID, deactivatedBy string) error

	// Usage tracking
	UpdateLastUsed(ctx context.Context, tenantID, moduleID string) error
	UpdateUsage(ctx context.Context, tenantID, moduleID string, usage map[string]any) error
}

// TenantToolRepository defines tenant tool activation operations
type TenantToolRepository interface {
	// CRUD operations
	Create(ctx context.Context, tenantTool *TenantTool) error
	GetByID(ctx context.Context, id string) (*TenantTool, error)
	GetByTenantAndTool(ctx context.Context, tenantID, toolID string) (*TenantTool, error)
	Update(ctx context.Context, tenantTool *TenantTool) error
	Delete(ctx context.Context, id string) error

	// Tenant-scoped queries
	ListByTenant(ctx context.Context, tenantID string, filters TenantToolFilters) ([]*TenantTool, error)
	ListActivatedByTenant(ctx context.Context, tenantID string) ([]*TenantTool, error)
	IsToolActivated(ctx context.Context, tenantID, toolID string) (bool, error)

	// Activation operations
	Activate(ctx context.Context, tenantID, toolID, activatedBy string) error
	Deactivate(ctx context.Context, tenantID, toolID, deactivatedBy string) error

	// Integration management
	VerifyIntegration(ctx context.Context, tenantID, toolID string) error
	UpdateIntegrationStatus(ctx context.Context, tenantID, toolID string, status IntegrationStatus) error

	// Health monitoring
	UpdateHealthStatus(ctx context.Context, tenantID, toolID string, status HealthStatus) error
	RecordError(ctx context.Context, tenantID, toolID, errorMsg string) error

	// Usage tracking
	UpdateLastUsed(ctx context.Context, tenantID, toolID string) error
	UpdateUsage(ctx context.Context, tenantID, toolID string, usage map[string]any) error
}

// Filter types
type ModuleFilters struct {
	Category     *ModuleCategory
	ModuleType   *ModuleType
	PricingModel *PricingModel
	Status       *ModuleStatus
	IsPublic     *bool
	IsBeta       *bool
	Tags         []string
	Limit        int
	Offset       int
	SortBy       string // "name", "install_count", "rating", "created_at"
	SortOrder    string // "asc", "desc"
}

type ToolFilters struct {
	Category     *ToolCategory
	ToolType     *ToolType
	Scope        *ToolScope
	PricingModel *PricingModel
	Status       *ToolStatus
	IsPublic     *bool
	IsBeta       *bool
	Tags         []string
	Limit        int
	Offset       int
	SortBy       string // "name", "install_count", "rating", "created_at"
	SortOrder    string // "asc", "desc"
}

type TenantModuleFilters struct {
	Status             *TenantModuleStatus
	IsEnabled          *bool
	SubscriptionStatus *SubscriptionStatus
	SetupCompleted     *bool
	Limit              int
	Offset             int
	SortBy             string // "installed_at", "activated_at", "last_used_at"
	SortOrder          string // "asc", "desc"
}

type TenantToolFilters struct {
	Status             *TenantToolStatus
	IsEnabled          *bool
	IntegrationStatus  *IntegrationStatus
	HealthStatus       *HealthStatus
	SubscriptionStatus *SubscriptionStatus
	SetupCompleted     *bool
	Limit              int
	Offset             int
	SortBy             string // "installed_at", "activated_at", "last_used_at"
	SortOrder          string // "asc", "desc"
}
