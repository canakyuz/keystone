package tenant

import "github.com/canakyuz/keystone/pkg/clienterr"

// Domain-specific errors for tenant operations
var (
	// Validation errors
	ErrInvalidTenantID          = clienterr.New("invalid tenant ID")
	ErrTenantNameRequired       = clienterr.New("tenant name is required")
	ErrInvalidTenantName        = clienterr.New("tenant name must be between 2 and 100 characters")
	ErrTenantSlugRequired       = clienterr.New("tenant slug is required")
	ErrInvalidTenantSlug        = clienterr.New("tenant slug must be between 2 and 50 characters")
	ErrTenantEmailRequired      = clienterr.New("tenant email is required")
	ErrInvalidTenantStatus      = clienterr.New("invalid tenant status")
	ErrInvalidSubscriptionPlan  = clienterr.New("invalid subscription plan")
	ErrTenantSchemaNameRequired = clienterr.New("tenant schema name is required")
	ErrInvalidTenantSchemaName  = clienterr.New("invalid tenant schema name")

	// Business logic errors
	ErrTenantNotFound         = clienterr.New("tenant not found")
	ErrTenantAlreadyExists    = clienterr.New("tenant already exists")
	ErrTenantSlugTaken        = clienterr.New("tenant slug is already taken")
	ErrTenantEmailTaken       = clienterr.New("tenant email is already registered")
	ErrTenantAlreadySuspended = clienterr.New("tenant is already suspended")
	ErrTenantAlreadyActive    = clienterr.New("tenant is already active")
	ErrTenantSuspended        = clienterr.New("tenant account is suspended")
	ErrTenantInactive         = clienterr.New("tenant account is inactive")
	ErrTrialExpired           = clienterr.New("trial period has expired")

	// Subscription errors
	ErrSameSubscriptionPlan = clienterr.New("tenant is already on this subscription plan")
	ErrInvalidPlanUpgrade   = clienterr.New("invalid plan upgrade")
	ErrSubscriptionExpired  = clienterr.New("subscription has expired")
	ErrFeatureNotAvailable  = clienterr.New("feature not available in current plan")
	ErrQuotaExceeded        = clienterr.New("quota exceeded for current plan")

	// Custom domain errors
	ErrInvalidCustomDomain         = clienterr.New("invalid custom domain")
	ErrCustomDomainNotSet          = clienterr.New("custom domain is not set")
	ErrCustomDomainAlreadyVerified = clienterr.New("custom domain is already verified")
	ErrCustomDomainNotVerified     = clienterr.New("custom domain is not verified")
	ErrCustomDomainTaken           = clienterr.New("custom domain is already in use")

	// Permission errors
	ErrTenantAccessDenied = clienterr.New("access denied to tenant resources")
	ErrCrossTenantAccess  = clienterr.New("cross-tenant access is not allowed")
)
