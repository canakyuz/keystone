package user

// Error is a domain error. Its message is written for the caller, so a transport may show
// it; an error that is not an *Error may carry driver or infrastructure detail and must
// not be shown.
type Error struct{ msg string }

// Error returns the message.
func (e *Error) Error() string { return e.msg }

func newError(msg string) *Error { return &Error{msg: msg} }

// Domain-specific errors for user operations
var (
	// Validation errors
	ErrInvalidUserID     = newError("invalid user ID")
	ErrTenantIDRequired  = newError("tenant ID is required")
	ErrEmailRequired     = newError("email is required")
	ErrInvalidEmail      = newError("invalid email format")
	ErrPasswordRequired  = newError("password is required")
	ErrPasswordTooShort  = newError("password must be at least 8 characters")
	ErrPasswordTooLong   = newError("password must not exceed 72 characters")
	ErrFirstNameRequired = newError("first name is required")
	ErrLastNameRequired  = newError("last name is required")
	ErrInvalidRole       = newError("invalid user role")
	ErrInvalidStatus     = newError("invalid user status")

	// Business logic errors
	ErrUserNotFound         = newError("user not found")
	ErrUserAlreadyExists    = newError("user already exists")
	ErrEmailAlreadyTaken    = newError("email is already registered")
	ErrUserAlreadyActive    = newError("user is already active")
	ErrUserAlreadySuspended = newError("user is already suspended")
	ErrUserSuspended        = newError("user account is suspended")
	ErrUserInactive         = newError("user account is inactive")
	ErrEmailNotVerified     = newError("email is not verified")
	ErrEmailAlreadyVerified = newError("email is already verified")
	ErrInvalidCredentials   = newError("invalid email or password")
	ErrSameRole             = newError("user already has this role")

	// Permission errors
	ErrUnauthorized            = newError("unauthorized access")
	ErrInsufficientPermissions = newError("insufficient permissions")
	ErrCannotModifyOwner       = newError("cannot modify owner account")
	ErrCannotDeleteSelf        = newError("cannot delete your own account")
	ErrCrossTenantUserAccess   = newError("cross-tenant user access is not allowed")

	// Two-factor authentication errors
	ErrTwoFactorRequired       = newError("two-factor authentication is required")
	ErrInvalidTwoFactorCode    = newError("invalid two-factor authentication code")
	ErrTwoFactorAlreadyEnabled = newError("two-factor authentication is already enabled")
	ErrTwoFactorNotEnabled     = newError("two-factor authentication is not enabled")
)
