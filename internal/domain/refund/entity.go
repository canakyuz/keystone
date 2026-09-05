package refund

import (
	"time"

	"github.com/google/uuid"
)

// RefundStatus represents the status of a refund
type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusSucceeded  RefundStatus = "succeeded"
	RefundStatusFailed     RefundStatus = "failed"
	RefundStatusCanceled   RefundStatus = "canceled"
)

// RefundReason represents the reason for refund
type RefundReason string

const (
	ReasonCustomerRequest RefundReason = "customer_request"
	ReasonDuplicate       RefundReason = "duplicate"
	ReasonFraudulent      RefundReason = "fraudulent"
	ReasonOther           RefundReason = "other"
)

// PaymentProvider represents different payment service providers
type PaymentProvider string

const (
	ProviderIyzico   PaymentProvider = "iyzico"
	ProviderCheckout PaymentProvider = "checkout"
	ProviderStripe   PaymentProvider = "stripe"
)

// Refund represents a payment refund transaction
type Refund struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	PaymentID string `json:"payment_id"`

	// Provider details
	Provider         PaymentProvider `json:"provider"`
	ProviderRefundID string          `json:"provider_refund_id,omitempty"`

	// Amount
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`

	// Status
	Status         RefundStatus `json:"status"`
	FailureCode    string       `json:"failure_code,omitempty"`
	FailureMessage string       `json:"failure_message,omitempty"`

	// Reason
	Reason      RefundReason `json:"reason"`
	Description string       `json:"description,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SucceededAt *time.Time `json:"succeeded_at,omitempty"`

	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// New creates a new refund with default values
func New(tenantID, paymentID string, provider PaymentProvider, amount float64, currency string, reason RefundReason) (*Refund, error) {
	now := time.Now()

	refund := &Refund{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		PaymentID: paymentID,
		Provider:  provider,
		Amount:    amount,
		Currency:  currency,
		Status:    RefundStatusPending,
		Reason:    reason,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}

	// Validate refund
	if err := refund.Validate(); err != nil {
		return nil, err
	}

	return refund, nil
}

// Validate validates refund data
func (r *Refund) Validate() error {
	if r.ID == "" {
		return ErrInvalidRefundID
	}

	if r.TenantID == "" {
		return ErrTenantIDRequired
	}

	if r.PaymentID == "" {
		return ErrPaymentIDRequired
	}

	if r.Amount <= 0 {
		return ErrInvalidAmount
	}

	if r.Currency == "" {
		return ErrCurrencyRequired
	}

	if !r.Provider.IsValid() {
		return ErrInvalidProvider
	}

	if !r.Status.IsValid() {
		return ErrInvalidRefundStatus
	}

	if !r.Reason.IsValid() {
		return ErrInvalidRefundReason
	}

	return nil
}

// MarkAsProcessing marks refund as processing
func (r *Refund) MarkAsProcessing() error {
	if r.Status != RefundStatusPending {
		return ErrInvalidStatusTransition
	}

	r.Status = RefundStatusProcessing
	r.UpdatedAt = time.Now()

	return nil
}

// MarkAsSucceeded marks refund as succeeded
func (r *Refund) MarkAsSucceeded(providerRefundID string) error {
	if r.Status != RefundStatusPending && r.Status != RefundStatusProcessing {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	r.Status = RefundStatusSucceeded
	r.ProviderRefundID = providerRefundID
	r.SucceededAt = &now
	r.UpdatedAt = now

	return nil
}

// MarkAsFailed marks refund as failed
func (r *Refund) MarkAsFailed(code, message string) error {
	if r.Status == RefundStatusSucceeded {
		return ErrInvalidStatusTransition
	}

	r.Status = RefundStatusFailed
	r.FailureCode = code
	r.FailureMessage = message
	r.UpdatedAt = time.Now()

	return nil
}

// MarkAsCanceled marks refund as canceled
func (r *Refund) MarkAsCanceled(reason string) error {
	if r.Status == RefundStatusSucceeded {
		return ErrInvalidStatusTransition
	}

	r.Status = RefundStatusCanceled
	r.UpdatedAt = time.Now()

	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata["cancelation_reason"] = reason
	r.Metadata["canceled_at"] = time.Now()

	return nil
}

// SetDescription sets refund description
func (r *Refund) SetDescription(description string) {
	r.Description = description
	r.UpdatedAt = time.Now()
}

// IsPending checks if refund is pending
func (r *Refund) IsPending() bool {
	return r.Status == RefundStatusPending
}

// IsProcessing checks if refund is processing
func (r *Refund) IsProcessing() bool {
	return r.Status == RefundStatusProcessing
}

// IsSucceeded checks if refund succeeded
func (r *Refund) IsSucceeded() bool {
	return r.Status == RefundStatusSucceeded
}

// IsFailed checks if refund failed
func (r *Refund) IsFailed() bool {
	return r.Status == RefundStatusFailed
}

// IsCanceled checks if refund is canceled
func (r *Refund) IsCanceled() bool {
	return r.Status == RefundStatusCanceled
}

// UpdateMetadata updates refund metadata
func (r *Refund) UpdateMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
	r.UpdatedAt = time.Now()
}

// GetMetadata retrieves refund metadata
func (r *Refund) GetMetadata(key string) (interface{}, bool) {
	if r.Metadata == nil {
		return nil, false
	}
	value, exists := r.Metadata[key]
	return value, exists
}

// IsValid checks if refund status is valid
func (s RefundStatus) IsValid() bool {
	switch s {
	case RefundStatusPending, RefundStatusProcessing, RefundStatusSucceeded,
		RefundStatusFailed, RefundStatusCanceled:
		return true
	default:
		return false
	}
}

// IsValid checks if refund reason is valid
func (r RefundReason) IsValid() bool {
	switch r {
	case ReasonCustomerRequest, ReasonDuplicate, ReasonFraudulent, ReasonOther:
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
