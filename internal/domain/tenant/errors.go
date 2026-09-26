package tenant

// Error is a domain error. Its message is written for the caller, so a transport may show
// it; an error that is not an *Error may carry driver or infrastructure detail and must
// not be shown.
type Error struct{ msg string }

// Error returns the message.
func (e *Error) Error() string { return e.msg }

func newError(msg string) *Error { return &Error{msg: msg} }

// Domain-specific errors for tenant operations
var (
	// Validation errors
	ErrInvalidTenantID          = newError("invalid tenant ID")
	ErrTenantNameRequired       = newError("tenant name is required")
	ErrInvalidTenantName        = newError("tenant name must be between 2 and 100 characters")
	ErrTenantSlugRequired       = newError("tenant slug is required")
	ErrInvalidTenantSlug        = newError("tenant slug must be between 2 and 50 characters")
	ErrTenantEmailRequired      = newError("tenant email is required")
	ErrInvalidTenantStatus      = newError("invalid tenant status")
	ErrInvalidSubscriptionPlan  = newError("invalid subscription plan")
	ErrTenantSchemaNameRequired = newError("tenant schema name is required")
	ErrInvalidTenantSchemaName  = newError("invalid tenant schema name")

	// Business logic errors
	ErrTenantNotFound         = newError("tenant not found")
	ErrTenantAlreadyExists    = newError("tenant already exists")
	ErrTenantSlugTaken        = newError("tenant slug is already taken")
	ErrTenantEmailTaken       = newError("tenant email is already registered")
	ErrTenantAlreadySuspended = newError("tenant is already suspended")
	ErrTenantAlreadyActive    = newError("tenant is already active")
	ErrTenantSuspended        = newError("tenant account is suspended")
	ErrTenantInactive         = newError("tenant account is inactive")
	ErrTrialExpired           = newError("trial period has expired")

	// Subscription errors
	ErrSameSubscriptionPlan = newError("tenant is already on this subscription plan")
	ErrInvalidPlanUpgrade   = newError("invalid plan upgrade")
	ErrSubscriptionExpired  = newError("subscription has expired")
	ErrFeatureNotAvailable  = newError("feature not available in current plan")
	ErrQuotaExceeded        = newError("quota exceeded for current plan")

	// Custom domain errors
	ErrInvalidCustomDomain         = newError("invalid custom domain")
	ErrCustomDomainNotSet          = newError("custom domain is not set")
	ErrCustomDomainAlreadyVerified = newError("custom domain is already verified")
	ErrCustomDomainNotVerified     = newError("custom domain is not verified")
	ErrCustomDomainTaken           = newError("custom domain is already in use")

	// Permission errors
	ErrTenantAccessDenied = newError("access denied to tenant resources")
	ErrCrossTenantAccess  = newError("cross-tenant access is not allowed")
)
