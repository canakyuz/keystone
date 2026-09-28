package payment

import "github.com/canakyuz/keystone/pkg/clienterr"

// Domain-specific errors for payment operations
var (
	// Validation errors
	ErrInvalidPaymentID       = clienterr.New("invalid payment ID")
	ErrTenantIDRequired       = clienterr.New("tenant ID is required")
	ErrInvalidAmount          = clienterr.New("amount must be greater than 0")
	ErrCurrencyRequired       = clienterr.New("currency is required")
	ErrInvalidProvider        = clienterr.New("invalid payment provider")
	ErrInvalidPaymentStatus   = clienterr.New("invalid payment status")
	ErrInvalidInstallment     = clienterr.New("installment must be at least 1")
	ErrInvalidInstallmentRate = clienterr.New("installment rate cannot be negative")

	// Business logic errors
	ErrPaymentNotFound         = clienterr.New("payment not found")
	ErrPaymentAlreadyExists    = clienterr.New("payment already exists")
	ErrInvalidStatusTransition = clienterr.New("invalid payment status transition")
	ErrPaymentNotSucceeded     = clienterr.New("payment has not succeeded")
	ErrPaymentAlreadyRefunded  = clienterr.New("payment is already refunded")
	ErrPaymentAlreadyCanceled  = clienterr.New("payment is already canceled")

	// 3DS errors
	Err3DSRequired         = clienterr.New("3DS authentication is required")
	ErrPaymentNot3DS       = clienterr.New("payment does not require 3DS authentication")
	Err3DSFailed           = clienterr.New("3DS authentication failed")
	Err3DSContentRequired  = clienterr.New("3DS HTML content is required")
	Err3DSCallbackRequired = clienterr.New("3DS callback URL is required")

	// Provider errors
	ErrProviderNotConfigured   = clienterr.New("payment provider is not configured")
	ErrProviderAPIError        = clienterr.New("payment provider API error")
	ErrProviderTimeout         = clienterr.New("payment provider request timeout")
	ErrProviderInvalidResponse = clienterr.New("invalid response from payment provider")

	// Card errors
	ErrInvalidCardNumber = clienterr.New("invalid card number")
	ErrInvalidCVV        = clienterr.New("invalid CVV")
	ErrInvalidExpiry     = clienterr.New("invalid card expiry")
	ErrCardDeclined      = clienterr.New("card was declined")
	ErrInsufficientFunds = clienterr.New("insufficient funds")
	ErrCardExpired       = clienterr.New("card has expired")

	// Refund errors
	ErrRefundNotAllowed    = clienterr.New("refund is not allowed for this payment")
	ErrRefundAmountInvalid = clienterr.New("refund amount is invalid")
	ErrRefundFailed        = clienterr.New("refund operation failed")

	// Permission errors
	ErrPaymentAccessDenied = clienterr.New("access denied to payment")
	ErrCrossTenantPayment  = clienterr.New("cross-tenant payment access is not allowed")
)
