package webhook

import "errors"

// Domain-specific errors for webhook operations
var (
	// Validation errors
	ErrInvalidEventID          = errors.New("invalid event ID")
	ErrTenantIDRequired        = errors.New("tenant ID is required")
	ErrProviderEventIDRequired = errors.New("provider event ID is required")
	ErrInvalidProvider         = errors.New("invalid payment provider")
	ErrInvalidEventType        = errors.New("invalid event type")
	ErrPayloadRequired         = errors.New("event payload is required")

	// Business logic errors
	ErrEventNotFound         = errors.New("event not found")
	ErrEventAlreadyProcessed = errors.New("event has already been processed")
	ErrEventProcessingFailed = errors.New("event processing failed")
	ErrMaxRetriesExceeded    = errors.New("maximum retry attempts exceeded")

	// Signature validation errors
	ErrInvalidSignature    = errors.New("invalid webhook signature")
	ErrSignatureNotFound   = errors.New("webhook signature not found")
	ErrSignatureExpired    = errors.New("webhook signature has expired")
	ErrInvalidSignatureAlg = errors.New("invalid signature algorithm")

	// Provider-specific errors
	ErrProviderNotConfigured   = errors.New("payment provider is not configured")
	ErrProviderAPIError        = errors.New("payment provider API error")
	ErrProviderInvalidResponse = errors.New("invalid response from payment provider")
	ErrUnsupportedEventType    = errors.New("unsupported event type")

	// Permission errors
	ErrEventAccessDenied = errors.New("access denied to event")
	ErrCrossTenantEvent  = errors.New("cross-tenant event access is not allowed")
)
