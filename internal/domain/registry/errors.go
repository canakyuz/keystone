package registry

import "errors"

// Module errors
var (
	ErrModuleNotFound        = errors.New("module not found")
	ErrInvalidModuleID       = errors.New("invalid module ID")
	ErrModuleNameRequired    = errors.New("module name is required")
	ErrInvalidModuleName     = errors.New("invalid module name (must be 2-100 characters)")
	ErrModuleSlugRequired    = errors.New("module slug is required")
	ErrInvalidModuleSlug     = errors.New("invalid module slug (must be 2-50 characters)")
	ErrModuleCodeRequired    = errors.New("module code is required")
	ErrInvalidModuleCode     = errors.New("invalid module code (must be 2-20 characters)")
	ErrInvalidModuleStatus   = errors.New("invalid module status")
	ErrInvalidModuleType     = errors.New("invalid module type")
	ErrInvalidPricingModel   = errors.New("invalid pricing model")
	ErrInvalidModuleCategory = errors.New("invalid module category")
	ErrInvalidRating         = errors.New("invalid rating (must be 0-5)")
	ErrInvalidPrice          = errors.New("invalid price (must be >= 0)")
)

// Tool errors
var (
	ErrToolNotFound           = errors.New("tool not found")
	ErrInvalidToolID          = errors.New("invalid tool ID")
	ErrToolNameRequired       = errors.New("tool name is required")
	ErrInvalidToolName        = errors.New("invalid tool name (must be 2-100 characters)")
	ErrToolSlugRequired       = errors.New("tool slug is required")
	ErrInvalidToolSlug        = errors.New("invalid tool slug (must be 2-50 characters)")
	ErrToolCodeRequired       = errors.New("tool code is required")
	ErrInvalidToolCode        = errors.New("invalid tool code (must be 2-20 characters)")
	ErrInvalidToolStatus      = errors.New("invalid tool status")
	ErrInvalidToolType        = errors.New("invalid tool type")
	ErrInvalidToolScope       = errors.New("invalid tool scope")
	ErrInvalidToolCategory    = errors.New("invalid tool category")
	ErrInvalidIntegrationType = errors.New("invalid integration type")
	ErrInvalidTransactionFee  = errors.New("invalid transaction fee (must be 0-100)")
)

// Tenant Module errors
var (
	ErrTenantModuleNotFound           = errors.New("tenant module not found")
	ErrInvalidTenantModuleID          = errors.New("invalid tenant module ID")
	ErrTenantIDRequired               = errors.New("tenant ID is required")
	ErrModuleIDRequired               = errors.New("module ID is required")
	ErrInvalidTenantModuleStatus      = errors.New("invalid tenant module status")
	ErrInvalidSubscriptionStatus      = errors.New("invalid subscription status")
	ErrTenantModuleAlreadyActive      = errors.New("tenant module already active")
	ErrTenantModuleAlreadyInactive    = errors.New("tenant module already inactive")
	ErrModuleNotActivated             = errors.New("module not activated for this tenant")
	ErrCannotDeactivateRequiredModule = errors.New("cannot deactivate required module")
)

// Tenant Tool errors
var (
	ErrTenantToolNotFound           = errors.New("tenant tool not found")
	ErrInvalidTenantToolID          = errors.New("invalid tenant tool ID")
	ErrToolIDRequired               = errors.New("tool ID is required")
	ErrInvalidTenantToolStatus      = errors.New("invalid tenant tool status")
	ErrInvalidIntegrationStatus     = errors.New("invalid integration status")
	ErrInvalidHealthStatus          = errors.New("invalid health status")
	ErrTenantToolAlreadyActive      = errors.New("tenant tool already active")
	ErrTenantToolAlreadyInactive    = errors.New("tenant tool already inactive")
	ErrToolNotActivated             = errors.New("tool not activated for this tenant")
	ErrCannotDeactivateRequiredTool = errors.New("cannot deactivate required tool")
	ErrIntegrationNotVerified       = errors.New("integration not verified")
)

// Dependency errors
var (
	ErrDependencyNotFound             = errors.New("dependency not found")
	ErrInvalidDependencyType          = errors.New("invalid dependency type")
	ErrInvalidDependencyScope         = errors.New("invalid dependency scope")
	ErrCircularDependency             = errors.New("circular dependency detected")
	ErrDependenciesNotMet             = errors.New("required dependencies not met")
	ErrMissingRequiredDependency      = errors.New("missing required dependency")
	ErrCannotDeactivateWithDependents = errors.New("cannot deactivate module/tool with active dependents")
)

// General errors
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrDatabaseError = errors.New("database error")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternalError = errors.New("internal error")
)
