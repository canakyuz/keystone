package user

import "errors"

// Domain-specific errors for user operations
var (
	// Validation errors
	ErrInvalidUserID      = errors.New("invalid user ID")
	ErrTenantIDRequired   = errors.New("tenant ID is required")
	ErrEmailRequired      = errors.New("email is required")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrPasswordRequired   = errors.New("password is required")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must not exceed 72 characters")
	ErrFirstNameRequired  = errors.New("first name is required")
	ErrLastNameRequired   = errors.New("last name is required")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrInvalidStatus      = errors.New("invalid user status")

	// Business logic errors
	ErrUserNotFound           = errors.New("user not found")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrEmailAlreadyTaken      = errors.New("email is already registered")
	ErrUserAlreadyActive      = errors.New("user is already active")
	ErrUserAlreadySuspended   = errors.New("user is already suspended")
	ErrUserSuspended          = errors.New("user account is suspended")
	ErrUserInactive           = errors.New("user account is inactive")
	ErrEmailNotVerified       = errors.New("email is not verified")
	ErrEmailAlreadyVerified   = errors.New("email is already verified")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrSameRole               = errors.New("user already has this role")

	// Permission errors
	ErrUnauthorized           = errors.New("unauthorized access")
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrCannotModifyOwner      = errors.New("cannot modify owner account")
	ErrCannotDeleteSelf       = errors.New("cannot delete your own account")
	ErrCrossTenantUserAccess  = errors.New("cross-tenant user access is not allowed")

	// Two-factor authentication errors
	ErrTwoFactorRequired      = errors.New("two-factor authentication is required")
	ErrInvalidTwoFactorCode   = errors.New("invalid two-factor authentication code")
	ErrTwoFactorAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	ErrTwoFactorNotEnabled    = errors.New("two-factor authentication is not enabled")
)
