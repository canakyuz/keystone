package webhook

import "github.com/canakyuz/keystone/pkg/clienterr"

// Domain-specific errors for webhook operations
var (
	// Validation errors
	ErrInvalidEventID          = clienterr.New("invalid event ID")
	ErrTenantIDRequired        = clienterr.New("tenant ID is required")
	ErrProviderEventIDRequired = clienterr.New("provider event ID is required")
	ErrInvalidProvider         = clienterr.New("invalid payment provider")
	ErrInvalidEventType        = clienterr.New("invalid event type")
	ErrPayloadRequired         = clienterr.New("event payload is required")

	// Business logic errors
	ErrEventNotFound         = clienterr.New("event not found")
	ErrEventAlreadyProcessed = clienterr.New("event has already been processed")
	ErrEventProcessingFailed = clienterr.New("event processing failed")
	ErrMaxRetriesExceeded    = clienterr.New("maximum retry attempts exceeded")

	// Signature validation errors
	ErrInvalidSignature    = clienterr.New("invalid webhook signature")
	ErrSignatureNotFound   = clienterr.New("webhook signature not found")
	ErrSignatureExpired    = clienterr.New("webhook signature has expired")
	ErrInvalidSignatureAlg = clienterr.New("invalid signature algorithm")

	// Provider-specific errors
	ErrProviderNotConfigured   = clienterr.New("payment provider is not configured")
	ErrProviderAPIError        = clienterr.New("payment provider API error")
	ErrProviderInvalidResponse = clienterr.New("invalid response from payment provider")
	ErrUnsupportedEventType    = clienterr.New("unsupported event type")

	// Permission errors
	ErrEventAccessDenied = clienterr.New("access denied to event")
	ErrCrossTenantEvent  = clienterr.New("cross-tenant event access is not allowed")
)
