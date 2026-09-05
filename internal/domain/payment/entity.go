package payment

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending     PaymentStatus = "pending"
	PaymentStatusProcessing  PaymentStatus = "processing"
	PaymentStatusRequires3DS PaymentStatus = "requires_3ds"
	PaymentStatusSucceeded   PaymentStatus = "succeeded"
	PaymentStatusFailed      PaymentStatus = "failed"
	PaymentStatusCanceled    PaymentStatus = "canceled"
	PaymentStatusRefunded    PaymentStatus = "refunded"
)

// PaymentProvider represents different payment service providers
type PaymentProvider string

const (
	ProviderIyzico   PaymentProvider = "iyzico"
	ProviderCheckout PaymentProvider = "checkout"
	ProviderStripe   PaymentProvider = "stripe"
)

// ThreeDSStatus represents 3DS authentication status
type ThreeDSStatus string

const (
	ThreeDSNotRequired ThreeDSStatus = "not_required"
	ThreeDSPending     ThreeDSStatus = "pending"
	ThreeDSSucceeded   ThreeDSStatus = "succeeded"
	ThreeDSFailed      ThreeDSStatus = "failed"
)

// CardBrand represents card brand
type CardBrand string

const (
	CardVisa       CardBrand = "visa"
	CardMastercard CardBrand = "mastercard"
	CardAmex       CardBrand = "amex"
	CardTroy       CardBrand = "troy"
	CardOther      CardBrand = "other"
)

// Payment represents a payment transaction
type Payment struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id,omitempty"`

	// Provider details
	Provider          PaymentProvider `json:"provider"`
	ProviderPaymentID string          `json:"provider_payment_id,omitempty"`

	// Amount
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`

	// Status
	Status         PaymentStatus `json:"status"`
	FailureCode    string        `json:"failure_code,omitempty"`
	FailureMessage string        `json:"failure_message,omitempty"`

	// 3DS (Secure Customer Authentication)
	Requires3DS        bool          `json:"requires_3ds"`
	ThreeDSVersion     string        `json:"threeds_version,omitempty"`
	ThreeDSHTMLContent string        `json:"threeds_html_content,omitempty"`
	ThreeDSCallbackURL string        `json:"threeds_callback_url,omitempty"`
	ThreeDSStatus      ThreeDSStatus `json:"threeds_status"`

	// Installment (Turkey specific)
	Installment     int     `json:"installment"`
	InstallmentRate float64 `json:"installment_rate"`

	// Card information
	CardBrand       CardBrand `json:"card_brand,omitempty"`
	CardLast4       string    `json:"card_last4,omitempty"`
	CardBIN         string    `json:"card_bin,omitempty"`
	CardExpMonth    string    `json:"card_exp_month,omitempty"`
	CardExpYear     string    `json:"card_exp_year,omitempty"`
	CardHolderName  string    `json:"card_holder_name,omitempty"`
	CardFingerprint string    `json:"card_fingerprint,omitempty"`

	// Customer information
	CustomerEmail     string `json:"customer_email,omitempty"`
	CustomerFirstName string `json:"customer_first_name,omitempty"`
	CustomerLastName  string `json:"customer_last_name,omitempty"`
	CustomerPhone     string `json:"customer_phone,omitempty"`
	CustomerIPAddress string `json:"customer_ip_address,omitempty"`

	// Billing address
	BillingCountry     string `json:"billing_country,omitempty"`
	BillingCity        string `json:"billing_city,omitempty"`
	BillingAddressLine string `json:"billing_address_line,omitempty"`
	BillingZipCode     string `json:"billing_zip_code,omitempty"`

	// Transaction details
	Description   string                 `json:"description,omitempty"`
	ReferenceID   string                 `json:"reference_id,omitempty"`
	OrderID       string                 `json:"order_id,omitempty"`
	InvoiceNumber string                 `json:"invoice_number,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SucceededAt *time.Time `json:"succeeded_at,omitempty"`
	CanceledAt  *time.Time `json:"canceled_at,omitempty"`

	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// New creates a new payment with default values
func New(tenantID string, provider PaymentProvider, amount float64, currency string) (*Payment, error) {
	now := time.Now()

	payment := &Payment{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Provider:    provider,
		Amount:      amount,
		Currency:    currency,
		Status:      PaymentStatusPending,
		Installment: 1,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    make(map[string]interface{}),
	}

	// Validate payment
	if err := payment.Validate(); err != nil {
		return nil, err
	}

	return payment, nil
}

// Validate validates payment data
func (p *Payment) Validate() error {
	if p.ID == "" {
		return ErrInvalidPaymentID
	}

	if p.TenantID == "" {
		return ErrTenantIDRequired
	}

	if p.Amount <= 0 {
		return ErrInvalidAmount
	}

	if p.Currency == "" {
		return ErrCurrencyRequired
	}

	if !p.Provider.IsValid() {
		return ErrInvalidProvider
	}

	if !p.Status.IsValid() {
		return ErrInvalidPaymentStatus
	}

	if p.Installment < 1 {
		return ErrInvalidInstallment
	}

	return nil
}

// MarkAsProcessing marks payment as processing
func (p *Payment) MarkAsProcessing() error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidStatusTransition
	}

	p.Status = PaymentStatusProcessing
	p.UpdatedAt = time.Now()

	return nil
}

// MarkAsRequires3DS marks payment as requiring 3DS authentication
func (p *Payment) MarkAsRequires3DS(threeDSVersion, htmlContent, callbackURL string) error {
	if p.Status != PaymentStatusPending && p.Status != PaymentStatusProcessing {
		return ErrInvalidStatusTransition
	}

	p.Status = PaymentStatusRequires3DS
	p.Requires3DS = true
	p.ThreeDSVersion = threeDSVersion
	p.ThreeDSHTMLContent = htmlContent
	p.ThreeDSCallbackURL = callbackURL
	p.ThreeDSStatus = ThreeDSPending
	p.UpdatedAt = time.Now()

	return nil
}

// Complete3DSAuthentication completes 3DS authentication
func (p *Payment) Complete3DSAuthentication(success bool) error {
	if p.Status != PaymentStatusRequires3DS {
		return ErrInvalidStatusTransition
	}

	if success {
		p.ThreeDSStatus = ThreeDSSucceeded
		p.Status = PaymentStatusProcessing
	} else {
		p.ThreeDSStatus = ThreeDSFailed
		p.Status = PaymentStatusFailed
		p.FailureCode = "3ds_failed"
		p.FailureMessage = "3DS authentication failed"
	}

	p.UpdatedAt = time.Now()

	return nil
}

// MarkAsSucceeded marks payment as succeeded
func (p *Payment) MarkAsSucceeded(providerPaymentID string) error {
	if p.Status != PaymentStatusPending && p.Status != PaymentStatusProcessing {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	p.Status = PaymentStatusSucceeded
	p.ProviderPaymentID = providerPaymentID
	p.SucceededAt = &now
	p.UpdatedAt = now

	return nil
}

// MarkAsFailed marks payment as failed
func (p *Payment) MarkAsFailed(code, message string) error {
	if p.Status == PaymentStatusSucceeded || p.Status == PaymentStatusRefunded {
		return ErrInvalidStatusTransition
	}

	p.Status = PaymentStatusFailed
	p.FailureCode = code
	p.FailureMessage = message
	p.UpdatedAt = time.Now()

	return nil
}

// MarkAsCanceled marks payment as canceled
func (p *Payment) MarkAsCanceled(reason string) error {
	if p.Status == PaymentStatusSucceeded || p.Status == PaymentStatusRefunded {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	p.Status = PaymentStatusCanceled
	p.CanceledAt = &now
	p.UpdatedAt = now

	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}
	p.Metadata["cancelation_reason"] = reason
	p.Metadata["canceled_at"] = now

	return nil
}

// MarkAsRefunded marks payment as refunded
func (p *Payment) MarkAsRefunded() error {
	if p.Status != PaymentStatusSucceeded {
		return ErrPaymentNotSucceeded
	}

	p.Status = PaymentStatusRefunded
	p.UpdatedAt = time.Now()

	return nil
}

// SetCardInfo sets card information (PCI-compliant, no full PAN)
func (p *Payment) SetCardInfo(brand CardBrand, last4, bin, expMonth, expYear, holderName, fingerprint string) {
	p.CardBrand = brand
	p.CardLast4 = last4
	p.CardBIN = bin
	p.CardExpMonth = expMonth
	p.CardExpYear = expYear
	p.CardHolderName = holderName
	p.CardFingerprint = fingerprint
	p.UpdatedAt = time.Now()
}

// SetCustomerInfo sets customer information
func (p *Payment) SetCustomerInfo(email, firstName, lastName, phone, ipAddress string) {
	p.CustomerEmail = email
	p.CustomerFirstName = firstName
	p.CustomerLastName = lastName
	p.CustomerPhone = phone
	p.CustomerIPAddress = ipAddress
	p.UpdatedAt = time.Now()
}

// SetBillingAddress sets billing address
func (p *Payment) SetBillingAddress(country, city, addressLine, zipCode string) {
	p.BillingCountry = country
	p.BillingCity = city
	p.BillingAddressLine = addressLine
	p.BillingZipCode = zipCode
	p.UpdatedAt = time.Now()
}

// SetInstallment sets installment options (Turkey specific)
func (p *Payment) SetInstallment(installment int, rate float64) error {
	if installment < 1 {
		return ErrInvalidInstallment
	}

	if rate < 0 {
		return ErrInvalidInstallmentRate
	}

	p.Installment = installment
	p.InstallmentRate = rate
	p.UpdatedAt = time.Now()

	return nil
}

// CalculateTotalAmount calculates total amount including installment fees
func (p *Payment) CalculateTotalAmount() float64 {
	if p.Installment <= 1 {
		return p.Amount
	}

	return p.Amount * (1 + p.InstallmentRate/100)
}

// IsPending checks if payment is pending
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsProcessing checks if payment is processing
func (p *Payment) IsProcessing() bool {
	return p.Status == PaymentStatusProcessing
}

// IsSucceeded checks if payment succeeded
func (p *Payment) IsSucceeded() bool {
	return p.Status == PaymentStatusSucceeded
}

// IsFailed checks if payment failed
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

// IsCanceled checks if payment is canceled
func (p *Payment) IsCanceled() bool {
	return p.Status == PaymentStatusCanceled
}

// IsRefunded checks if payment is refunded
func (p *Payment) IsRefunded() bool {
	return p.Status == PaymentStatusRefunded
}

// Requires3DSAuth checks if payment requires 3DS authentication
func (p *Payment) Requires3DSAuth() bool {
	return p.Requires3DS && p.Status == PaymentStatusRequires3DS
}

// CanBeRefunded checks if payment can be refunded
func (p *Payment) CanBeRefunded() bool {
	return p.Status == PaymentStatusSucceeded
}

// CanBeCanceled checks if payment can be canceled
func (p *Payment) CanBeCanceled() bool {
	return p.Status == PaymentStatusPending || p.Status == PaymentStatusProcessing
}

// UpdateMetadata updates payment metadata
func (p *Payment) UpdateMetadata(key string, value interface{}) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}
	p.Metadata[key] = value
	p.UpdatedAt = time.Now()
}

// GetMetadata retrieves payment metadata
func (p *Payment) GetMetadata(key string) (interface{}, bool) {
	if p.Metadata == nil {
		return nil, false
	}
	value, exists := p.Metadata[key]
	return value, exists
}

// IsValid checks if payment status is valid
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusProcessing, PaymentStatusRequires3DS,
		PaymentStatusSucceeded, PaymentStatusFailed, PaymentStatusCanceled, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

// IsValid checks if payment provider is valid
func (p PaymentProvider) IsValid() bool {
	switch p {
	case ProviderIyzico, ProviderCheckout, ProviderStripe:
		return true
	default:
		return false
	}
}

// IsValid checks if 3DS status is valid
func (s ThreeDSStatus) IsValid() bool {
	switch s {
	case ThreeDSNotRequired, ThreeDSPending, ThreeDSSucceeded, ThreeDSFailed:
		return true
	default:
		return false
	}
}

// IsValid checks if card brand is valid
func (b CardBrand) IsValid() bool {
	switch b {
	case CardVisa, CardMastercard, CardAmex, CardTroy, CardOther:
		return true
	default:
		return false
	}
}
