package user

import "github.com/canakyuz/keystone/pkg/clienterr"

// Domain-specific errors for user operations
var (
	// Validation errors
	ErrInvalidUserID     = clienterr.New("invalid user ID")
	ErrTenantIDRequired  = clienterr.New("tenant ID is required")
	ErrEmailRequired     = clienterr.New("email is required")
	ErrInvalidEmail      = clienterr.New("invalid email format")
	ErrPasswordRequired  = clienterr.New("password is required")
	ErrPasswordTooShort  = clienterr.New("password must be at least 8 characters")
	ErrPasswordTooLong   = clienterr.New("password must not exceed 72 characters")
	ErrFirstNameRequired = clienterr.New("first name is required")
	ErrLastNameRequired  = clienterr.New("last name is required")
	ErrInvalidRole       = clienterr.New("invalid user role")
	ErrInvalidStatus     = clienterr.New("invalid user status")

	// Business logic errors
	ErrUserNotFound         = clienterr.New("user not found")
	ErrUserAlreadyExists    = clienterr.New("user already exists")
	ErrEmailAlreadyTaken    = clienterr.New("email is already registered")
	ErrUserAlreadyActive    = clienterr.New("user is already active")
	ErrUserAlreadySuspended = clienterr.New("user is already suspended")
	ErrUserSuspended        = clienterr.New("user account is suspended")
	ErrUserInactive         = clienterr.New("user account is inactive")
	ErrEmailNotVerified     = clienterr.New("email is not verified")
	ErrEmailAlreadyVerified = clienterr.New("email is already verified")
	ErrInvalidCredentials   = clienterr.New("invalid email or password")
	ErrSameRole             = clienterr.New("user already has this role")

	// Permission errors
	ErrUnauthorized            = clienterr.New("unauthorized access")
	ErrInsufficientPermissions = clienterr.New("insufficient permissions")
	ErrCannotModifyOwner       = clienterr.New("cannot modify owner account")
	ErrCannotDeleteSelf        = clienterr.New("cannot delete your own account")
	ErrCrossTenantUserAccess   = clienterr.New("cross-tenant user access is not allowed")

	// Two-factor authentication errors
	ErrTwoFactorRequired       = clienterr.New("two-factor authentication is required")
	ErrInvalidTwoFactorCode    = clienterr.New("invalid two-factor authentication code")
	ErrTwoFactorAlreadyEnabled = clienterr.New("two-factor authentication is already enabled")
	ErrTwoFactorNotEnabled     = clienterr.New("two-factor authentication is not enabled")
)
