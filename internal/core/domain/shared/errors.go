package shared

import (
	"errors"
	"fmt"
)

// Domain errors - these represent business rule violations
var (
	// Generic errors
	ErrNotFound            = errors.New("entity not found")
	ErrAlreadyExists       = errors.New("entity already exists")
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrForbidden           = errors.New("forbidden operation")
	ErrInvalidOperation    = errors.New("invalid operation")
	ErrConcurrencyConflict = errors.New("concurrency conflict")

	// Tenant-specific errors
	ErrTenantNotFound       = errors.New("tenant not found")
	ErrTenantSlugExists     = errors.New("tenant slug already exists")
	ErrTenantNotActive      = errors.New("tenant is not active")
	ErrTenantSuspended      = errors.New("tenant is suspended")
	ErrTenantLimitExceeded  = errors.New("tenant limit exceeded")
	ErrInvalidTenantContext = errors.New("invalid tenant context")

	// User-specific errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserEmailExists    = errors.New("user email already exists")
	ErrUserNotActive      = errors.New("user is not active")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPasswordTooWeak    = errors.New("password too weak")
	ErrInvalidRole        = errors.New("invalid user role")

	// Permission errors
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrInvalidPermission       = errors.New("invalid permission")
	ErrPermissionNotFound      = errors.New("permission not found")

	// Template/Module errors
	ErrTemplateNotFound         = errors.New("template not found")
	ErrTemplateNotPublic        = errors.New("template is not public")
	ErrTemplateAlreadyInstalled = errors.New("template already installed")
	ErrModuleNotFound           = errors.New("module not found")
	ErrModuleNotCompatible      = errors.New("module not compatible")

	// Subscription/Billing errors
	ErrSubscriptionNotFound    = errors.New("subscription not found")
	ErrSubscriptionExpired     = errors.New("subscription expired")
	ErrInvalidSubscriptionPlan = errors.New("invalid subscription plan")
	ErrPaymentRequired         = errors.New("payment required")
	ErrInsufficientFunds       = errors.New("insufficient funds")

	// Business rule violations
	ErrBusinessRuleViolation = errors.New("business rule violation")
	ErrQuotaExceeded         = errors.New("quota exceeded")
	ErrFeatureNotAvailable   = errors.New("feature not available")
)

// DomainError wraps domain-specific errors with additional context
type DomainError struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %s)", e.Type, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap implements error unwrapping for Go 1.13+
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// AddDetail adds contextual information to the error
func (e *DomainError) AddDetail(key string, value interface{}) *DomainError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewDomainError creates a new domain error
func NewDomainError(errType, message string) *DomainError {
	return &DomainError{
		Type:    errType,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WrapDomainError wraps an existing error with domain context
func WrapDomainError(err error, errType, message string) *DomainError {
	return &DomainError{
		Type:    errType,
		Message: message,
		Cause:   err,
		Details: make(map[string]interface{}),
	}
}

// Validation Error types
const (
	ValidationError   = "validation_error"
	BusinessRuleError = "business_rule_error"
	PermissionError   = "permission_error"
	NotFoundError     = "not_found_error"
	ConflictError     = "conflict_error"
	QuotaError        = "quota_error"
	SubscriptionError = "subscription_error"
)

// Common domain error constructors
func NewValidationError(message string) *DomainError {
	return NewDomainError(ValidationError, message)
}

func NewBusinessRuleError(message string) *DomainError {
	return NewDomainError(BusinessRuleError, message)
}

func NewPermissionError(message string) *DomainError {
	return NewDomainError(PermissionError, message)
}

func NewNotFoundError(entity string) *DomainError {
	return NewDomainError(NotFoundError, fmt.Sprintf("%s not found", entity))
}

func NewConflictError(message string) *DomainError {
	return NewDomainError(ConflictError, message)
}

func NewQuotaError(resource string) *DomainError {
	return NewDomainError(QuotaError, fmt.Sprintf("%s quota exceeded", resource))
}

func NewSubscriptionError(message string) *DomainError {
	return NewDomainError(SubscriptionError, message)
}
