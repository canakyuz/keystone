package tenant

import "errors"

// Domain-specific errors for tenant operations
var (
	// Validation errors
	ErrInvalidTenantID       = errors.New("invalid tenant ID")
	ErrTenantNameRequired    = errors.New("tenant name is required")
	ErrInvalidTenantName     = errors.New("tenant name must be between 2 and 100 characters")
	ErrTenantSlugRequired    = errors.New("tenant slug is required")
	ErrInvalidTenantSlug     = errors.New("tenant slug must be between 2 and 50 characters")
	ErrTenantEmailRequired   = errors.New("tenant email is required")
	ErrInvalidTenantStatus   = errors.New("invalid tenant status")
	ErrInvalidSubscriptionPlan = errors.New("invalid subscription plan")

	// Business logic errors
	ErrTenantNotFound         = errors.New("tenant not found")
	ErrTenantAlreadyExists    = errors.New("tenant already exists")
	ErrTenantSlugTaken        = errors.New("tenant slug is already taken")
	ErrTenantEmailTaken       = errors.New("tenant email is already registered")
	ErrTenantAlreadySuspended = errors.New("tenant is already suspended")
	ErrTenantAlreadyActive    = errors.New("tenant is already active")
	ErrTenantSuspended        = errors.New("tenant account is suspended")
	ErrTenantInactive         = errors.New("tenant account is inactive")
	ErrTrialExpired           = errors.New("trial period has expired")

	// Subscription errors
	ErrSameSubscriptionPlan   = errors.New("tenant is already on this subscription plan")
	ErrInvalidPlanUpgrade     = errors.New("invalid plan upgrade")
	ErrSubscriptionExpired    = errors.New("subscription has expired")
	ErrFeatureNotAvailable    = errors.New("feature not available in current plan")
	ErrQuotaExceeded          = errors.New("quota exceeded for current plan")

	// Custom domain errors
	ErrInvalidCustomDomain         = errors.New("invalid custom domain")
	ErrCustomDomainNotSet          = errors.New("custom domain is not set")
	ErrCustomDomainAlreadyVerified = errors.New("custom domain is already verified")
	ErrCustomDomainNotVerified     = errors.New("custom domain is not verified")
	ErrCustomDomainTaken           = errors.New("custom domain is already in use")

	// Permission errors
	ErrTenantAccessDenied = errors.New("access denied to tenant resources")
	ErrCrossTenantAccess  = errors.New("cross-tenant access is not allowed")
)
