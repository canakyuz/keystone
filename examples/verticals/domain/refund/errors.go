package refund

import "errors"

// Domain-specific errors for refund operations
var (
	// Validation errors
	ErrInvalidRefundID     = errors.New("invalid refund ID")
	ErrTenantIDRequired    = errors.New("tenant ID is required")
	ErrPaymentIDRequired   = errors.New("payment ID is required")
	ErrInvalidAmount       = errors.New("refund amount must be greater than 0")
	ErrCurrencyRequired    = errors.New("currency is required")
	ErrInvalidProvider     = errors.New("invalid payment provider")
	ErrInvalidRefundStatus = errors.New("invalid refund status")
	ErrInvalidRefundReason = errors.New("invalid refund reason")

	// Business logic errors
	ErrRefundNotFound          = errors.New("refund not found")
	ErrRefundAlreadyExists     = errors.New("refund already exists")
	ErrInvalidStatusTransition = errors.New("invalid refund status transition")
	ErrRefundAmountExceeded    = errors.New("refund amount exceeds payment amount")
	ErrPaymentNotRefundable    = errors.New("payment is not refundable")
	ErrPartialRefundNotAllowed = errors.New("partial refund is not allowed")

	// Provider errors
	ErrProviderNotConfigured   = errors.New("payment provider is not configured")
	ErrProviderAPIError        = errors.New("payment provider API error")
	ErrProviderTimeout         = errors.New("payment provider request timeout")
	ErrProviderInvalidResponse = errors.New("invalid response from payment provider")
	ErrRefundNotSupported      = errors.New("refund is not supported by provider")

	// Permission errors
	ErrRefundAccessDenied = errors.New("access denied to refund")
	ErrCrossTenantRefund  = errors.New("cross-tenant refund access is not allowed")
)
