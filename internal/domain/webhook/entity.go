package webhook

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents different webhook event types
type EventType string

const (
	// Payment events
	EventPaymentSucceeded EventType = "payment.succeeded"
	EventPaymentFailed    EventType = "payment.failed"
	EventPaymentPending   EventType = "payment.pending"
	EventPaymentCanceled  EventType = "payment.canceled"

	// Refund events
	EventRefundSucceeded EventType = "refund.succeeded"
	EventRefundFailed    EventType = "refund.failed"
	EventRefundPending   EventType = "refund.pending"

	// 3DS events
	Event3DSRequired  EventType = "3ds.required"
	Event3DSSucceeded EventType = "3ds.succeeded"
	Event3DSFailed    EventType = "3ds.failed"

	// Chargeback events
	EventChargebackCreated EventType = "chargeback.created"
	EventChargebackUpdated EventType = "chargeback.updated"
)

// PaymentProvider represents different payment service providers
type PaymentProvider string

const (
	ProviderIyzico   PaymentProvider = "iyzico"
	ProviderCheckout PaymentProvider = "checkout"
	ProviderStripe   PaymentProvider = "stripe"
)

// PaymentEvent represents a webhook event from payment providers
type PaymentEvent struct {
	ID       string    `json:"id"`
	TenantID string    `json:"tenant_id"`

	// Provider details
	Provider        PaymentProvider `json:"provider"`
	ProviderEventID string          `json:"provider_event_id"` // For idempotency

	// Event data
	EventType    EventType              `json:"event_type"`
	EventVersion string                 `json:"event_version,omitempty"`
	Payload      map[string]interface{} `json:"payload"` // Full webhook payload

	// Related resources
	PaymentID *string `json:"payment_id,omitempty"`
	RefundID  *string `json:"refund_id,omitempty"`

	// Processing status
	Processed       bool       `json:"processed"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ProcessingError string     `json:"processing_error,omitempty"`
	RetryCount      int        `json:"retry_count"`

	// Request metadata
	RequestIPAddress string                 `json:"request_ip_address,omitempty"`
	RequestHeaders   map[string]interface{} `json:"request_headers,omitempty"`
	SignatureValid   bool                   `json:"signature_valid"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// New creates a new payment event
func New(tenantID string, provider PaymentProvider, providerEventID string, eventType EventType, payload map[string]interface{}) (*PaymentEvent, error) {
	now := time.Now()

	event := &PaymentEvent{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Provider:        provider,
		ProviderEventID: providerEventID,
		EventType:       eventType,
		Payload:         payload,
		Processed:       false,
		RetryCount:      0,
		SignatureValid:  false,
		CreatedAt:       now,
		UpdatedAt:       now,
		RequestHeaders:  make(map[string]interface{}),
	}

	// Validate event
	if err := event.Validate(); err != nil {
		return nil, err
	}

	return event, nil
}

// Validate validates payment event data
func (e *PaymentEvent) Validate() error {
	if e.ID == "" {
		return ErrInvalidEventID
	}

	if e.TenantID == "" {
		return ErrTenantIDRequired
	}

	if e.ProviderEventID == "" {
		return ErrProviderEventIDRequired
	}

	if !e.Provider.IsValid() {
		return ErrInvalidProvider
	}

	if !e.EventType.IsValid() {
		return ErrInvalidEventType
	}

	if e.Payload == nil || len(e.Payload) == 0 {
		return ErrPayloadRequired
	}

	return nil
}

// MarkAsProcessed marks the event as processed
func (e *PaymentEvent) MarkAsProcessed() error {
	if e.Processed {
		return ErrEventAlreadyProcessed
	}

	now := time.Now()
	e.Processed = true
	e.ProcessedAt = &now
	e.UpdatedAt = now
	e.ProcessingError = ""

	return nil
}

// MarkAsFailed marks the event processing as failed
func (e *PaymentEvent) MarkAsFailed(errorMsg string) error {
	e.Processed = false
	e.ProcessingError = errorMsg
	e.RetryCount++
	e.UpdatedAt = time.Now()

	return nil
}

// SetSignatureValid marks signature as valid
func (e *PaymentEvent) SetSignatureValid(valid bool) {
	e.SignatureValid = valid
	e.UpdatedAt = time.Now()
}

// SetPaymentID associates event with payment
func (e *PaymentEvent) SetPaymentID(paymentID string) {
	e.PaymentID = &paymentID
	e.UpdatedAt = time.Now()
}

// SetRefundID associates event with refund
func (e *PaymentEvent) SetRefundID(refundID string) {
	e.RefundID = &refundID
	e.UpdatedAt = time.Now()
}

// SetRequestMetadata sets request metadata
func (e *PaymentEvent) SetRequestMetadata(ipAddress string, headers map[string]interface{}) {
	e.RequestIPAddress = ipAddress
	e.RequestHeaders = headers
	e.UpdatedAt = time.Now()
}

// IsProcessed checks if event is processed
func (e *PaymentEvent) IsProcessed() bool {
	return e.Processed
}

// NeedsRetry checks if event needs retry
func (e *PaymentEvent) NeedsRetry(maxRetries int) bool {
	return !e.Processed && e.RetryCount < maxRetries
}

// IsSignatureValid checks if signature is valid
func (e *PaymentEvent) IsSignatureValid() bool {
	return e.SignatureValid
}

// IsPaymentEvent checks if event is related to payment
func (e *PaymentEvent) IsPaymentEvent() bool {
	switch e.EventType {
	case EventPaymentSucceeded, EventPaymentFailed, EventPaymentPending, EventPaymentCanceled:
		return true
	default:
		return false
	}
}

// IsRefundEvent checks if event is related to refund
func (e *PaymentEvent) IsRefundEvent() bool {
	switch e.EventType {
	case EventRefundSucceeded, EventRefundFailed, EventRefundPending:
		return true
	default:
		return false
	}
}

// Is3DSEvent checks if event is related to 3DS
func (e *PaymentEvent) Is3DSEvent() bool {
	switch e.EventType {
	case Event3DSRequired, Event3DSSucceeded, Event3DSFailed:
		return true
	default:
		return false
	}
}

// IsChargebackEvent checks if event is related to chargeback
func (e *PaymentEvent) IsChargebackEvent() bool {
	switch e.EventType {
	case EventChargebackCreated, EventChargebackUpdated:
		return true
	default:
		return false
	}
}

// GetPayloadValue retrieves a value from the payload
func (e *PaymentEvent) GetPayloadValue(key string) (interface{}, bool) {
	if e.Payload == nil {
		return nil, false
	}
	value, exists := e.Payload[key]
	return value, exists
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

// IsValid checks if event type is valid
func (t EventType) IsValid() bool {
	switch t {
	case EventPaymentSucceeded, EventPaymentFailed, EventPaymentPending, EventPaymentCanceled,
		EventRefundSucceeded, EventRefundFailed, EventRefundPending,
		Event3DSRequired, Event3DSSucceeded, Event3DSFailed,
		EventChargebackCreated, EventChargebackUpdated:
		return true
	default:
		return false
	}
}

// String returns string representation of event type
func (t EventType) String() string {
	return string(t)
}

// String returns string representation of provider
func (p PaymentProvider) String() string {
	return string(p)
}
