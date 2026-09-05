package payment

import (
	"time"

	"github.com/canakyuz/keystone/internal/domain/payment"
	"github.com/canakyuz/keystone/internal/domain/refund"
	"github.com/canakyuz/keystone/internal/domain/webhook"
)

// CreatePaymentRequest represents a payment creation request
type CreatePaymentRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Currency    string  `json:"currency" validate:"required,len=3"`
	Description string  `json:"description" validate:"required,max=500"`
	OrderID     string  `json:"order_id,omitempty"`

	// Customer information
	CustomerEmail     string `json:"customer_email" validate:"required,email"`
	CustomerFirstName string `json:"customer_first_name" validate:"required"`
	CustomerLastName  string `json:"customer_last_name" validate:"required"`
	CustomerPhone     string `json:"customer_phone,omitempty"`

	// Card information (tokenized or direct)
	CardToken      string `json:"card_token,omitempty"`
	CardNumber     string `json:"card_number,omitempty"`
	CardExpMonth   string `json:"card_exp_month,omitempty"`
	CardExpYear    string `json:"card_exp_year,omitempty"`
	CardCVV        string `json:"card_cvv,omitempty"`
	CardHolderName string `json:"card_holder_name,omitempty"`

	// Billing address
	BillingCountry     string `json:"billing_country" validate:"required"`
	BillingCity        string `json:"billing_city" validate:"required"`
	BillingAddressLine string `json:"billing_address_line" validate:"required"`
	BillingZipCode     string `json:"billing_zip_code,omitempty"`

	// Installment (Turkey specific)
	Installment int `json:"installment" validate:"omitempty,gte=1,lte=12"`

	// URLs for 3DS
	SuccessURL  string `json:"success_url,omitempty"`
	FailureURL  string `json:"failure_url,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Complete3DSRequest represents 3DS completion request
type Complete3DSRequest struct {
	PaymentID        string            `json:"payment_id" validate:"required,uuid"`
	ConversationData string            `json:"conversation_data,omitempty"`
	CallbackParams   map[string]string `json:"callback_params,omitempty"`
}

// CreateRefundRequest represents a refund creation request
type CreateRefundRequest struct {
	PaymentID   string  `json:"payment_id" validate:"required,uuid"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Reason      string  `json:"reason" validate:"required,oneof=customer_request duplicate fraudulent other"`
	Description string  `json:"description,omitempty"`
}

// PaymentResponse represents payment response DTO
type PaymentResponse struct {
	ID                string  `json:"id"`
	TenantID          string  `json:"tenant_id"`
	Provider          string  `json:"provider"`
	ProviderPaymentID string  `json:"provider_payment_id,omitempty"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Status            string  `json:"status"`
	FailureCode       string  `json:"failure_code,omitempty"`
	FailureMessage    string  `json:"failure_message,omitempty"`

	// 3DS
	Requires3DS        bool   `json:"requires_3ds"`
	ThreeDSHTMLContent string `json:"threeds_html_content,omitempty"`
	ThreeDSRedirectURL string `json:"threeds_redirect_url,omitempty"`

	// Card info (masked)
	CardBrand    string `json:"card_brand,omitempty"`
	CardLast4    string `json:"card_last4,omitempty"`
	CardExpMonth string `json:"card_exp_month,omitempty"`
	CardExpYear  string `json:"card_exp_year,omitempty"`

	// Installment
	Installment     int     `json:"installment,omitempty"`
	InstallmentRate float64 `json:"installment_rate,omitempty"`
	TotalAmount     float64 `json:"total_amount,omitempty"`

	// Customer
	CustomerEmail string `json:"customer_email,omitempty"`

	// Metadata
	Description string                 `json:"description,omitempty"`
	OrderID     string                 `json:"order_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SucceededAt *time.Time `json:"succeeded_at,omitempty"`
}

// RefundResponse represents refund response DTO
type RefundResponse struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	PaymentID        string                 `json:"payment_id"`
	Provider         string                 `json:"provider"`
	ProviderRefundID string                 `json:"provider_refund_id,omitempty"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	Status           string                 `json:"status"`
	FailureCode      string                 `json:"failure_code,omitempty"`
	FailureMessage   string                 `json:"failure_message,omitempty"`
	Reason           string                 `json:"reason"`
	Description      string                 `json:"description,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	SucceededAt      *time.Time             `json:"succeeded_at,omitempty"`
}

// WebhookEventResponse represents webhook event response DTO
type WebhookEventResponse struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	Provider        string                 `json:"provider"`
	ProviderEventID string                 `json:"provider_event_id"`
	EventType       string                 `json:"event_type"`
	EventVersion    string                 `json:"event_version,omitempty"`
	PaymentID       *string                `json:"payment_id,omitempty"`
	RefundID        *string                `json:"refund_id,omitempty"`
	Processed       bool                   `json:"processed"`
	ProcessedAt     *time.Time             `json:"processed_at,omitempty"`
	ProcessingError string                 `json:"processing_error,omitempty"`
	RetryCount      int                    `json:"retry_count"`
	SignatureValid  bool                   `json:"signature_valid"`
	Payload         map[string]interface{} `json:"payload,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ListPaymentsRequest represents list payments request
type ListPaymentsRequest struct {
	Limit  int `json:"limit" validate:"omitempty,gte=1,lte=100"`
	Offset int `json:"offset" validate:"omitempty,gte=0"`
}

// ToPaymentResponse converts domain payment to DTO
func ToPaymentResponse(p *payment.Payment) *PaymentResponse {
	resp := &PaymentResponse{
		ID:                 p.ID,
		TenantID:           p.TenantID,
		Provider:           string(p.Provider),
		ProviderPaymentID:  p.ProviderPaymentID,
		Amount:             p.Amount,
		Currency:           p.Currency,
		Status:             string(p.Status),
		FailureCode:        p.FailureCode,
		FailureMessage:     p.FailureMessage,
		Requires3DS:        p.Requires3DS,
		ThreeDSHTMLContent: p.ThreeDSHTMLContent,
		CardBrand:          string(p.CardBrand),
		CardLast4:          p.CardLast4,
		CardExpMonth:       p.CardExpMonth,
		CardExpYear:        p.CardExpYear,
		Installment:        p.Installment,
		InstallmentRate:    p.InstallmentRate,
		TotalAmount:        p.CalculateTotalAmount(),
		CustomerEmail:      p.CustomerEmail,
		Description:        p.Description,
		OrderID:            p.OrderID,
		Metadata:           p.Metadata,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		SucceededAt:        p.SucceededAt,
	}

	return resp
}

// ToRefundResponse converts domain refund to DTO
func ToRefundResponse(r *refund.Refund) *RefundResponse {
	return &RefundResponse{
		ID:               r.ID,
		TenantID:         r.TenantID,
		PaymentID:        r.PaymentID,
		Provider:         string(r.Provider),
		ProviderRefundID: r.ProviderRefundID,
		Amount:           r.Amount,
		Currency:         r.Currency,
		Status:           string(r.Status),
		FailureCode:      r.FailureCode,
		FailureMessage:   r.FailureMessage,
		Reason:           string(r.Reason),
		Description:      r.Description,
		Metadata:         r.Metadata,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		SucceededAt:      r.SucceededAt,
	}
}

// ToWebhookEventResponse converts domain webhook event to DTO
func ToWebhookEventResponse(e *webhook.PaymentEvent) *WebhookEventResponse {
	return &WebhookEventResponse{
		ID:              e.ID,
		TenantID:        e.TenantID,
		Provider:        string(e.Provider),
		ProviderEventID: e.ProviderEventID,
		EventType:       string(e.EventType),
		EventVersion:    e.EventVersion,
		PaymentID:       e.PaymentID,
		RefundID:        e.RefundID,
		Processed:       e.Processed,
		ProcessedAt:     e.ProcessedAt,
		ProcessingError: e.ProcessingError,
		RetryCount:      e.RetryCount,
		SignatureValid:  e.SignatureValid,
		Payload:         e.Payload,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}
