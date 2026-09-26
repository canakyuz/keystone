package registry

// ModuleFilters narrows a module catalogue query. A nil field is not filtered on, a zero
// Limit takes the default page size, and an empty SortBy the default order.
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

// ToolFilters narrows a tool catalogue query, with the conventions of ModuleFilters.
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

// TenantModuleFilters narrows the modules installed for one tenant, with the conventions
// of ModuleFilters.
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

// TenantToolFilters narrows the tools installed for one tenant, with the conventions of
// ModuleFilters.
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
