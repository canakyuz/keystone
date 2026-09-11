package payment

import (
	"context"
)

// Provider defines the interface that all payment providers must implement
type Provider interface {
	// GetName returns the provider name (iyzico, checkout, stripe)
	GetName() string

	// CreatePayment initiates a payment transaction
	CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error)

	// CompletePayment completes a payment (for 3DS callbacks)
	CompletePayment(ctx context.Context, req *CompletePaymentRequest) (*PaymentResponse, error)

	// GetPayment retrieves payment details from provider
	GetPayment(ctx context.Context, providerPaymentID string) (*PaymentResponse, error)

	// CancelPayment cancels a pending payment
	CancelPayment(ctx context.Context, providerPaymentID string) error

	// CreateRefund initiates a refund transaction
	CreateRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)

	// GetRefund retrieves refund details from provider
	GetRefund(ctx context.Context, providerRefundID string) (*RefundResponse, error)

	// VerifyWebhookSignature verifies webhook signature
	VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error

	// ParseWebhook parses webhook payload into standard event
	ParseWebhook(ctx context.Context, payload []byte) (*WebhookEvent, error)
}

// PaymentRequest represents a payment creation request
type PaymentRequest struct {
	TenantID    string
	Amount      float64
	Currency    string
	Description string
	OrderID     string
	ReferenceID string

	// Customer information
	CustomerEmail     string
	CustomerFirstName string
	CustomerLastName  string
	CustomerPhone     string
	CustomerIPAddress string

	// Card information (PCI-compliant tokenized)
	CardToken      string
	CardNumber     string // Only for providers that support direct card
	CardExpMonth   string
	CardExpYear    string
	CardCVV        string
	CardHolderName string

	// Billing address
	BillingCountry     string
	BillingCity        string
	BillingAddressLine string
	BillingZipCode     string

	// Installment (Turkey specific)
	Installment int

	// 3DS settings
	CallbackURL string
	FailureURL  string
	SuccessURL  string

	// Additional metadata
	Metadata map[string]any
}

// PaymentResponse represents a payment response from provider
type PaymentResponse struct {
	ProviderPaymentID string
	Status            string // pending, processing, requires_3ds, succeeded, failed
	Amount            float64
	Currency          string

	// 3DS authentication
	Requires3DS        bool
	ThreeDSVersion     string
	ThreeDSHTMLContent string
	ThreeDSRedirectURL string

	// Card information (masked)
	CardBrand       string
	CardLast4       string
	CardBIN         string
	CardExpMonth    string
	CardExpYear     string
	CardFingerprint string

	// Installment details
	Installment     int
	InstallmentRate float64

	// Failure details
	FailureCode    string
	FailureMessage string

	// Provider raw response
	RawResponse map[string]any
}

// CompletePaymentRequest represents a 3DS completion request
type CompletePaymentRequest struct {
	ProviderPaymentID string
	PaymentID         string // Our internal payment ID
	ConversationData  string // 3DS conversation data
	CallbackParams    map[string]string
}

// RefundRequest represents a refund creation request
type RefundRequest struct {
	TenantID          string
	PaymentID         string
	ProviderPaymentID string
	Amount            float64
	Currency          string
	Reason            string
	Description       string
	Metadata          map[string]any
}

// RefundResponse represents a refund response from provider
type RefundResponse struct {
	ProviderRefundID string
	Status           string // pending, processing, succeeded, failed
	Amount           float64
	Currency         string
	FailureCode      string
	FailureMessage   string
	RawResponse      map[string]any
}

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	ProviderEventID string
	EventType       string // payment.succeeded, payment.failed, refund.succeeded, etc.
	EventVersion    string
	PaymentID       string
	RefundID        string
	Status          string
	Amount          float64
	Currency        string
	FailureCode     string
	FailureMessage  string
	Payload         map[string]any
}

// InstallmentOption represents an installment plan option
type InstallmentOption struct {
	Installment int     `json:"installment"`
	Rate        float64 `json:"rate"`
	TotalAmount float64 `json:"total_amount"`
}

// GetInstallmentOptions returns available installment options
type GetInstallmentOptions interface {
	GetInstallmentOptions(ctx context.Context, amount float64, currency string, cardBIN string) ([]InstallmentOption, error)
}
