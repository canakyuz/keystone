package errors

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"nexspaces-api/internal/core/domain/shared"
)

// HTTPError represents an HTTP error with additional context
type HTTPError struct {
	Status  int         `json:"status"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Cause   error       `json:"-"`
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("HTTP %d: %s (caused by: %v)", e.Status, e.Message, e.Cause)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(status int, message string, cause error) *HTTPError {
	return &HTTPError{
		Status:  status,
		Code:    getErrorCode(status),
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError creates a validation error from validator errors
func NewValidationError(message string, err error) *HTTPError {
	var validationErrors []ValidationField

	if validatorErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validatorErrs {
			validationErrors = append(validationErrors, ValidationField{
				Field:   fieldErr.Field(),
				Tag:     fieldErr.Tag(),
				Value:   fieldErr.Value(),
				Message: getValidationMessage(fieldErr),
			})
		}
	}

	return &HTTPError{
		Status:  fiber.StatusBadRequest,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: ValidationErrorDetails{
			Fields: validationErrors,
		},
		Cause: err,
	}
}

// ValidationField represents a field validation error
type ValidationField struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}

// ValidationErrorDetails contains validation error details
type ValidationErrorDetails struct {
	Fields []ValidationField `json:"fields"`
}

// HandleDomainError converts domain errors to HTTP errors
func HandleDomainError(err error) *HTTPError {
	// Handle specific domain errors
	switch {
	case errors.Is(err, shared.ErrTenantNotFound):
		return NewHTTPError(fiber.StatusNotFound, "Tenant not found", err)
	case errors.Is(err, shared.ErrTenantSlugAlreadyExists):
		return NewHTTPError(fiber.StatusConflict, "Tenant slug already exists", err)
	case errors.Is(err, shared.ErrTenantNotActive):
		return NewHTTPError(fiber.StatusForbidden, "Tenant is not active", err)
	case errors.Is(err, shared.ErrCustomDomainAlreadyExists):
		return NewHTTPError(fiber.StatusConflict, "Custom domain already exists", err)

	case errors.Is(err, shared.ErrUserNotFound):
		return NewHTTPError(fiber.StatusNotFound, "User not found", err)
	case errors.Is(err, shared.ErrUserAlreadyExists):
		return NewHTTPError(fiber.StatusConflict, "User already exists", err)
	case errors.Is(err, shared.ErrInvalidCredentials):
		return NewHTTPError(fiber.StatusUnauthorized, "Invalid credentials", err)
	case errors.Is(err, shared.ErrUserNotActive):
		return NewHTTPError(fiber.StatusForbidden, "User is not active", err)
	case errors.Is(err, shared.ErrEmailNotVerified):
		return NewHTTPError(fiber.StatusForbidden, "Email not verified", err)
	case errors.Is(err, shared.ErrCannotDeactivateSelf):
		return NewHTTPError(fiber.StatusBadRequest, "Cannot deactivate yourself", err)

	case errors.Is(err, shared.ErrTemplateNotFound):
		return NewHTTPError(fiber.StatusNotFound, "Template not found", err)
	case errors.Is(err, shared.ErrTemplateAlreadyExists):
		return NewHTTPError(fiber.StatusConflict, "Template already exists", err)
	case errors.Is(err, shared.ErrTemplateNotAccessible):
		return NewHTTPError(fiber.StatusForbidden, "Template not accessible", err)
	case errors.Is(err, shared.ErrTemplateAlreadyInstalled):
		return NewHTTPError(fiber.StatusConflict, "Template already installed", err)
	case errors.Is(err, shared.ErrInstallationNotFound):
		return NewHTTPError(fiber.StatusNotFound, "Installation not found", err)

	case errors.Is(err, shared.ErrSubscriptionNotFound):
		return NewHTTPError(fiber.StatusNotFound, "Subscription not found", err)
	case errors.Is(err, shared.ErrTenantAlreadyHasSubscription):
		return NewHTTPError(fiber.StatusConflict, "Tenant already has an active subscription", err)
	case errors.Is(err, shared.ErrSubscriptionNotActive):
		return NewHTTPError(fiber.StatusForbidden, "Subscription is not active", err)
	case errors.Is(err, shared.ErrUsageLimitExceeded):
		return NewHTTPError(fiber.StatusForbidden, "Usage limit exceeded", err)
	case errors.Is(err, shared.ErrPaymentRequired):
		return NewHTTPError(fiber.StatusPaymentRequired, "Payment required", err)

	case errors.Is(err, shared.ErrCrossTenantAccess):
		return NewHTTPError(fiber.StatusForbidden, "Cross-tenant access denied", err)
	case errors.Is(err, shared.ErrInsufficientPermissions):
		return NewHTTPError(fiber.StatusForbidden, "Insufficient permissions", err)
	case errors.Is(err, shared.ErrResourceLocked):
		return NewHTTPError(fiber.StatusLocked, "Resource is locked", err)
	case errors.Is(err, shared.ErrRateLimitExceeded):
		return NewHTTPError(fiber.StatusTooManyRequests, "Rate limit exceeded", err)

	default:
		// For unknown errors, return internal server error without exposing details
		return NewHTTPError(fiber.StatusInternalServerError, "Internal server error", err)
	}
}

// getErrorCode returns an error code based on HTTP status
func getErrorCode(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusConflict:
		return "CONFLICT"
	case fiber.StatusTooManyRequests:
		return "RATE_LIMITED"
	case fiber.StatusInternalServerError:
		return "INTERNAL_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}

// getValidationMessage returns a human-readable validation error message
func getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be no more than %s characters long", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	case "slug":
		return fmt.Sprintf("%s must be a valid slug (lowercase, alphanumeric, hyphens)", fe.Field())
	case "fqdn":
		return fmt.Sprintf("%s must be a valid domain name", fe.Field())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   *HTTPError `json:"error"`
}

// ToErrorResponse converts an HTTPError to the standard error response format
func ToErrorResponse(err *HTTPError) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   err,
	}
}
