package refund

import "github.com/canakyuz/keystone/pkg/clienterr"

// Domain-specific errors for refund operations
var (
	// Validation errors
	ErrInvalidRefundID     = clienterr.New("invalid refund ID")
	ErrTenantIDRequired    = clienterr.New("tenant ID is required")
	ErrPaymentIDRequired   = clienterr.New("payment ID is required")
	ErrInvalidAmount       = clienterr.New("refund amount must be greater than 0")
	ErrCurrencyRequired    = clienterr.New("currency is required")
	ErrInvalidProvider     = clienterr.New("invalid payment provider")
	ErrInvalidRefundStatus = clienterr.New("invalid refund status")
	ErrInvalidRefundReason = clienterr.New("invalid refund reason")

	// Business logic errors
	ErrRefundNotFound          = clienterr.New("refund not found")
	ErrRefundAlreadyExists     = clienterr.New("refund already exists")
	ErrInvalidStatusTransition = clienterr.New("invalid refund status transition")
	ErrRefundAmountExceeded    = clienterr.New("refund amount exceeds payment amount")
	ErrPaymentNotRefundable    = clienterr.New("payment is not refundable")
	ErrPartialRefundNotAllowed = clienterr.New("partial refund is not allowed")

	// Provider errors
	ErrProviderNotConfigured   = clienterr.New("payment provider is not configured")
	ErrProviderAPIError        = clienterr.New("payment provider API error")
	ErrProviderTimeout         = clienterr.New("payment provider request timeout")
	ErrProviderInvalidResponse = clienterr.New("invalid response from payment provider")
	ErrRefundNotSupported      = clienterr.New("refund is not supported by provider")

	// Permission errors
	ErrRefundAccessDenied = clienterr.New("access denied to refund")
	ErrCrossTenantRefund  = clienterr.New("cross-tenant refund access is not allowed")
)
