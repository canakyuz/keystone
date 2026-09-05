package payment

import "errors"

// Domain-specific errors for payment operations
var (
	// Validation errors
	ErrInvalidPaymentID       = errors.New("invalid payment ID")
	ErrTenantIDRequired       = errors.New("tenant ID is required")
	ErrInvalidAmount          = errors.New("amount must be greater than 0")
	ErrCurrencyRequired       = errors.New("currency is required")
	ErrInvalidProvider        = errors.New("invalid payment provider")
	ErrInvalidPaymentStatus   = errors.New("invalid payment status")
	ErrInvalidInstallment     = errors.New("installment must be at least 1")
	ErrInvalidInstallmentRate = errors.New("installment rate cannot be negative")

	// Business logic errors
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrPaymentAlreadyExists    = errors.New("payment already exists")
	ErrInvalidStatusTransition = errors.New("invalid payment status transition")
	ErrPaymentNotSucceeded     = errors.New("payment has not succeeded")
	ErrPaymentAlreadyRefunded  = errors.New("payment is already refunded")
	ErrPaymentAlreadyCanceled  = errors.New("payment is already canceled")

	// 3DS errors
	Err3DSRequired         = errors.New("3DS authentication is required")
	Err3DSFailed           = errors.New("3DS authentication failed")
	Err3DSContentRequired  = errors.New("3DS HTML content is required")
	Err3DSCallbackRequired = errors.New("3DS callback URL is required")

	// Provider errors
	ErrProviderNotConfigured   = errors.New("payment provider is not configured")
	ErrProviderAPIError        = errors.New("payment provider API error")
	ErrProviderTimeout         = errors.New("payment provider request timeout")
	ErrProviderInvalidResponse = errors.New("invalid response from payment provider")

	// Card errors
	ErrInvalidCardNumber = errors.New("invalid card number")
	ErrInvalidCVV        = errors.New("invalid CVV")
	ErrInvalidExpiry     = errors.New("invalid card expiry")
	ErrCardDeclined      = errors.New("card was declined")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrCardExpired       = errors.New("card has expired")

	// Refund errors
	ErrRefundNotAllowed    = errors.New("refund is not allowed for this payment")
	ErrRefundAmountInvalid = errors.New("refund amount is invalid")
	ErrRefundFailed        = errors.New("refund operation failed")

	// Permission errors
	ErrPaymentAccessDenied = errors.New("access denied to payment")
	ErrCrossTenantPayment  = errors.New("cross-tenant payment access is not allowed")
)
