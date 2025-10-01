package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents application-specific error codes
type ErrorCode string

const (
	// General errors
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeBadRequest     ErrorCode = "BAD_REQUEST"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeRateLimited    ErrorCode = "RATE_LIMITED"

	// Authentication errors
	ErrCodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrCodeExpiredToken       ErrorCode = "EXPIRED_TOKEN"
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"

	// Multi-tenant errors
	ErrCodeTenantNotFound     ErrorCode = "TENANT_NOT_FOUND"
	ErrCodeTenantMismatch     ErrorCode = "TENANT_MISMATCH"
	ErrCodeCrossTenantAccess  ErrorCode = "CROSS_TENANT_ACCESS"
	ErrCodeTenantSuspended    ErrorCode = "TENANT_SUSPENDED"

	// Resource errors
	ErrCodeResourceNotFound   ErrorCode = "RESOURCE_NOT_FOUND"
	ErrCodeResourceExists     ErrorCode = "RESOURCE_ALREADY_EXISTS"
	ErrCodeResourceInUse      ErrorCode = "RESOURCE_IN_USE"

	// Database errors
	ErrCodeDatabaseConnection ErrorCode = "DATABASE_CONNECTION_ERROR"
	ErrCodeDatabaseQuery      ErrorCode = "DATABASE_QUERY_ERROR"
	ErrCodeDatabaseConstraint ErrorCode = "DATABASE_CONSTRAINT_VIOLATION"

	// Business logic errors
	ErrCodeInvalidOperation   ErrorCode = "INVALID_OPERATION"
	ErrCodeQuotaExceeded      ErrorCode = "QUOTA_EXCEEDED"
	ErrCodeFeatureDisabled    ErrorCode = "FEATURE_DISABLED"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
	Err        error                  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap implements error unwrapping
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds additional context to the error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: codeToHTTPStatus(code),
	}
}

// Wrap wraps an existing error with AppError context
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: codeToHTTPStatus(code),
		Err:        err,
	}
}

// Is checks if an error is of a specific type
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As attempts to convert error to specific type
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// codeToHTTPStatus maps error codes to HTTP status codes
func codeToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeNotFound, ErrCodeResourceNotFound, ErrCodeTenantNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized, ErrCodeInvalidToken, ErrCodeExpiredToken, ErrCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrCodeForbidden, ErrCodeCrossTenantAccess, ErrCodeTenantMismatch:
		return http.StatusForbidden
	case ErrCodeBadRequest, ErrCodeValidation, ErrCodeInvalidOperation:
		return http.StatusBadRequest
	case ErrCodeConflict, ErrCodeResourceExists:
		return http.StatusConflict
	case ErrCodeRateLimited, ErrCodeQuotaExceeded:
		return http.StatusTooManyRequests
	case ErrCodeTenantSuspended, ErrCodeFeatureDisabled:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// Common error constructors for convenience

// ErrNotFound creates a not found error
func ErrNotFound(resource string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return New(ErrCodeUnauthorized, message)
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return New(ErrCodeForbidden, message)
}

// ErrBadRequest creates a bad request error
func ErrBadRequest(message string) *AppError {
	return New(ErrCodeBadRequest, message)
}

// ErrValidation creates a validation error
func ErrValidation(field, reason string) *AppError {
	return New(ErrCodeValidation, fmt.Sprintf("Validation failed for field '%s': %s", field, reason)).
		WithDetails("field", field).
		WithDetails("reason", reason)
}

// ErrConflict creates a conflict error
func ErrConflict(resource string) *AppError {
	return New(ErrCodeConflict, fmt.Sprintf("%s already exists", resource))
}

// ErrInternal creates an internal server error
func ErrInternal(message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return New(ErrCodeInternal, message)
}

// ErrTenantNotFound creates a tenant not found error
func ErrTenantNotFound(tenantID string) *AppError {
	return New(ErrCodeTenantNotFound, "Tenant not found").
		WithDetails("tenant_id", tenantID)
}

// ErrTenantMismatch creates a tenant mismatch error
func ErrTenantMismatch(expectedTenantID, actualTenantID string) *AppError {
	return New(ErrCodeTenantMismatch, "Tenant mismatch detected").
		WithDetails("expected_tenant_id", expectedTenantID).
		WithDetails("actual_tenant_id", actualTenantID)
}

// ErrCrossTenantAccess creates a cross-tenant access error
func ErrCrossTenantAccess() *AppError {
	return New(ErrCodeCrossTenantAccess, "Cross-tenant access is not allowed")
}

// ErrTenantSuspended creates a tenant suspended error
func ErrTenantSuspended(tenantID string) *AppError {
	return New(ErrCodeTenantSuspended, "Tenant account is suspended").
		WithDetails("tenant_id", tenantID)
}

// ErrInvalidToken creates an invalid token error
func ErrInvalidToken() *AppError {
	return New(ErrCodeInvalidToken, "Invalid authentication token")
}

// ErrExpiredToken creates an expired token error
func ErrExpiredToken() *AppError {
	return New(ErrCodeExpiredToken, "Authentication token has expired")
}

// ErrInvalidCredentials creates an invalid credentials error
func ErrInvalidCredentials() *AppError {
	return New(ErrCodeInvalidCredentials, "Invalid email or password")
}

// ErrQuotaExceeded creates a quota exceeded error
func ErrQuotaExceeded(resource string, limit int) *AppError {
	return New(ErrCodeQuotaExceeded, fmt.Sprintf("Quota exceeded for %s", resource)).
		WithDetails("resource", resource).
		WithDetails("limit", limit)
}

// ErrFeatureDisabled creates a feature disabled error
func ErrFeatureDisabled(feature string) *AppError {
	return New(ErrCodeFeatureDisabled, fmt.Sprintf("Feature '%s' is not enabled for your plan", feature)).
		WithDetails("feature", feature)
}

// ErrDatabaseConnection creates a database connection error
func ErrDatabaseConnection(err error) *AppError {
	return Wrap(err, ErrCodeDatabaseConnection, "Failed to connect to database")
}

// ErrDatabaseQuery creates a database query error
func ErrDatabaseQuery(err error, query string) *AppError {
	return Wrap(err, ErrCodeDatabaseQuery, "Database query failed").
		WithDetails("query", query)
}

// ValidationError represents a collection of validation errors
type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

// Error implements the error interface for ValidationError
func (v *ValidationError) Error() string {
	return "Validation failed"
}

// Add adds a field validation error
func (v *ValidationError) Add(field, message string) {
	if v.Fields == nil {
		v.Fields = make(map[string]string)
	}
	v.Fields[field] = message
}

// HasErrors checks if there are any validation errors
func (v *ValidationError) HasErrors() bool {
	return len(v.Fields) > 0
}

// ToAppError converts ValidationError to AppError
func (v *ValidationError) ToAppError() *AppError {
	appErr := New(ErrCodeValidation, "Validation failed")
	for field, msg := range v.Fields {
		appErr.WithDetails(field, msg)
	}
	return appErr
}

// NewValidationError creates a new ValidationError
func NewValidationError() *ValidationError {
	return &ValidationError{
		Fields: make(map[string]string),
	}
}
