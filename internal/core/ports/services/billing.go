package services

import (
	"context"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/subscription"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// BillingService defines the interface for billing operations
type BillingService interface {
	// CreateSubscription creates a new subscription
	CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*subscription.Subscription, error)

	// CancelSubscription cancels subscription
	CancelSubscription(ctx context.Context, tenantID tenant.TenantID, reason string) error

	// ReactivateSubscription reactivates canceled subscription
	ReactivateSubscription(ctx context.Context, tenantID tenant.TenantID) error

	// ChangeSubscriptionPlan changes subscription plan
	ChangeSubscriptionPlan(ctx context.Context, tenantID tenant.TenantID, newPlanID subscription.PlanID) error

	// ChangeBillingCycle changes billing cycle
	ChangeBillingCycle(ctx context.Context, tenantID tenant.TenantID, cycle subscription.BillingCycle) error

	// ProcessPayment processes subscription payment
	ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*PaymentResult, error)

	// HandleFailedPayment handles failed payment
	HandleFailedPayment(ctx context.Context, subscriptionID subscription.SubscriptionID, reason string) error

	// RefundPayment refunds payment
	RefundPayment(ctx context.Context, paymentID string, amount *shared.Money, reason string) (*RefundResult, error)

	// GetSubscription returns subscription for tenant
	GetSubscription(ctx context.Context, tenantID tenant.TenantID) (*subscription.Subscription, error)

	// GetUpcomingInvoice returns upcoming invoice
	GetUpcomingInvoice(ctx context.Context, tenantID tenant.TenantID) (*Invoice, error)

	// GetInvoices returns invoices for tenant
	GetInvoices(ctx context.Context, tenantID tenant.TenantID, filter InvoiceFilter) ([]*Invoice, error)

	// GetPaymentMethods returns payment methods for tenant
	GetPaymentMethods(ctx context.Context, tenantID tenant.TenantID) ([]*PaymentMethod, error)

	// AddPaymentMethod adds payment method
	AddPaymentMethod(ctx context.Context, tenantID tenant.TenantID, req AddPaymentMethodRequest) (*PaymentMethod, error)

	// RemovePaymentMethod removes payment method
	RemovePaymentMethod(ctx context.Context, tenantID tenant.TenantID, methodID string) error

	// SetDefaultPaymentMethod sets default payment method
	SetDefaultPaymentMethod(ctx context.Context, tenantID tenant.TenantID, methodID string) error

	// UpdateUsage updates subscription usage
	UpdateUsage(ctx context.Context, tenantID tenant.TenantID, usage subscription.Usage) error

	// GetUsage returns current usage
	GetUsage(ctx context.Context, tenantID tenant.TenantID) (*subscription.Usage, error)

	// CheckUsageLimits checks if usage exceeds limits
	CheckUsageLimits(ctx context.Context, tenantID tenant.TenantID) (*UsageLimitStatus, error)

	// GetBillingPortalURL returns URL to billing portal
	GetBillingPortalURL(ctx context.Context, tenantID tenant.TenantID, returnURL string) (string, error)

	// WebhookHandler handles payment provider webhooks
	WebhookHandler(ctx context.Context, provider string, payload []byte, signature string) error
}

// PaymentProviderService defines the interface for payment provider operations
type PaymentProviderService interface {
	// CreateCustomer creates customer in payment provider
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*Customer, error)

	// UpdateCustomer updates customer information
	UpdateCustomer(ctx context.Context, customerID string, req UpdateCustomerRequest) (*Customer, error)

	// DeleteCustomer deletes customer
	DeleteCustomer(ctx context.Context, customerID string) error

	// CreateSubscription creates subscription in payment provider
	CreateSubscription(ctx context.Context, req CreateProviderSubscriptionRequest) (*ProviderSubscription, error)

	// UpdateSubscription updates subscription
	UpdateSubscription(ctx context.Context, subscriptionID string, req UpdateProviderSubscriptionRequest) (*ProviderSubscription, error)

	// CancelSubscription cancels subscription
	CancelSubscription(ctx context.Context, subscriptionID string, cancelAt *shared.Timestamp) error

	// ProcessPayment processes one-time payment
	ProcessPayment(ctx context.Context, req ProcessProviderPaymentRequest) (*ProviderPayment, error)

	// CreatePaymentIntent creates payment intent
	CreatePaymentIntent(ctx context.Context, req CreatePaymentIntentRequest) (*PaymentIntent, error)

	// ConfirmPaymentIntent confirms payment intent
	ConfirmPaymentIntent(ctx context.Context, intentID string, paymentMethodID string) (*PaymentIntent, error)

	// RefundPayment refunds payment
	RefundPayment(ctx context.Context, paymentID string, amount *shared.Money, reason string) (*Refund, error)

	// GetCustomer retrieves customer
	GetCustomer(ctx context.Context, customerID string) (*Customer, error)

	// GetSubscription retrieves subscription
	GetSubscription(ctx context.Context, subscriptionID string) (*ProviderSubscription, error)

	// GetPayment retrieves payment
	GetPayment(ctx context.Context, paymentID string) (*ProviderPayment, error)

	// AddPaymentMethod adds payment method
	AddPaymentMethod(ctx context.Context, customerID string, req AddProviderPaymentMethodRequest) (*ProviderPaymentMethod, error)

	// DetachPaymentMethod detaches payment method
	DetachPaymentMethod(ctx context.Context, methodID string) error

	// ListPaymentMethods lists customer payment methods
	ListPaymentMethods(ctx context.Context, customerID string) ([]*ProviderPaymentMethod, error)

	// GetInvoices retrieves invoices
	GetInvoices(ctx context.Context, customerID string, filter ProviderInvoiceFilter) ([]*ProviderInvoice, error)

	// CreateBillingPortalSession creates billing portal session
	CreateBillingPortalSession(ctx context.Context, customerID string, returnURL string) (*BillingPortalSession, error)

	// ConstructWebhookEvent constructs webhook event from payload
	ConstructWebhookEvent(ctx context.Context, payload []byte, signature string, secret string) (*WebhookEvent, error)
}

// Request/Response types

// CreateSubscriptionRequest represents subscription creation request
type CreateSubscriptionRequest struct {
	TenantID        tenant.TenantID           `json:"tenant_id"`
	PlanID          subscription.PlanID       `json:"plan_id"`
	BillingCycle    subscription.BillingCycle `json:"billing_cycle"`
	PaymentMethodID string                    `json:"payment_method_id"`
	StartTrial      bool                      `json:"start_trial"`
	CouponCode      *string                   `json:"coupon_code"`
	Metadata        map[string]interface{}    `json:"metadata"`
}

// ProcessPaymentRequest represents payment processing request
type ProcessPaymentRequest struct {
	SubscriptionID  subscription.SubscriptionID `json:"subscription_id"`
	Amount          shared.Money                `json:"amount"`
	PaymentMethodID string                      `json:"payment_method_id"`
	Description     string                      `json:"description"`
	Metadata        map[string]interface{}      `json:"metadata"`
}

// PaymentResult represents payment processing result
type PaymentResult struct {
	PaymentID     string                 `json:"payment_id"`
	Status        PaymentStatus          `json:"status"`
	Amount        shared.Money           `json:"amount"`
	Currency      string                 `json:"currency"`
	TransactionID string                 `json:"transaction_id"`
	ProcessedAt   shared.Timestamp       `json:"processed_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// RefundResult represents refund result
type RefundResult struct {
	RefundID    string           `json:"refund_id"`
	Status      RefundStatus     `json:"status"`
	Amount      shared.Money     `json:"amount"`
	Reason      string           `json:"reason"`
	ProcessedAt shared.Timestamp `json:"processed_at"`
}

// RefundStatus represents refund status
type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "pending"
	RefundStatusSucceeded RefundStatus = "succeeded"
	RefundStatusFailed    RefundStatus = "failed"
	RefundStatusCanceled  RefundStatus = "canceled"
)

// Invoice represents an invoice
type Invoice struct {
	ID               string                      `json:"id"`
	TenantID         tenant.TenantID             `json:"tenant_id"`
	SubscriptionID   subscription.SubscriptionID `json:"subscription_id"`
	Number           string                      `json:"number"`
	Status           InvoiceStatus               `json:"status"`
	Amount           shared.Money                `json:"amount"`
	AmountPaid       shared.Money                `json:"amount_paid"`
	AmountDue        shared.Money                `json:"amount_due"`
	Currency         string                      `json:"currency"`
	Description      string                      `json:"description"`
	LineItems        []InvoiceLineItem           `json:"line_items"`
	TaxAmount        *shared.Money               `json:"tax_amount"`
	DiscountAmount   *shared.Money               `json:"discount_amount"`
	PeriodStart      shared.Timestamp            `json:"period_start"`
	PeriodEnd        shared.Timestamp            `json:"period_end"`
	DueDate          shared.Timestamp            `json:"due_date"`
	PaidAt           *shared.Timestamp           `json:"paid_at"`
	VoidedAt         *shared.Timestamp           `json:"voided_at"`
	DownloadURL      string                      `json:"download_url"`
	HostedInvoiceURL string                      `json:"hosted_invoice_url"`
	Metadata         map[string]interface{}      `json:"metadata"`
	CreatedAt        shared.Timestamp            `json:"created_at"`
}

// InvoiceStatus represents invoice status
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid          InvoiceStatus = "void"
)

// InvoiceLineItem represents invoice line item
type InvoiceLineItem struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Amount      shared.Money           `json:"amount"`
	Quantity    int                    `json:"quantity"`
	UnitAmount  shared.Money           `json:"unit_amount"`
	Type        string                 `json:"type"` // subscription, one_time, usage
	PeriodStart *shared.Timestamp      `json:"period_start"`
	PeriodEnd   *shared.Timestamp      `json:"period_end"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// InvoiceFilter represents invoice filtering options
type InvoiceFilter struct {
	Status    *InvoiceStatus
	DueAfter  *shared.Timestamp
	DueBefore *shared.Timestamp
	Limit     int
	Offset    int
}

// PaymentMethod represents a payment method
type PaymentMethod struct {
	ID          string                 `json:"id"`
	Type        PaymentMethodType      `json:"type"`
	Card        *CardDetails           `json:"card,omitempty"`
	BankAccount *BankAccountDetails    `json:"bank_account,omitempty"`
	IsDefault   bool                   `json:"is_default"`
	ExpiresAt   *shared.Timestamp      `json:"expires_at"`
	CreatedAt   shared.Timestamp       `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PaymentMethodType represents payment method type
type PaymentMethodType string

const (
	PaymentMethodCard        PaymentMethodType = "card"
	PaymentMethodBankAccount PaymentMethodType = "bank_account"
	PaymentMethodWallet      PaymentMethodType = "wallet"
)

// CardDetails represents card details
type CardDetails struct {
	Brand       string `json:"brand"`
	Last4       string `json:"last4"`
	ExpMonth    int    `json:"exp_month"`
	ExpYear     int    `json:"exp_year"`
	Fingerprint string `json:"fingerprint"`
	Country     string `json:"country"`
}

// BankAccountDetails represents bank account details
type BankAccountDetails struct {
	BankName      string `json:"bank_name"`
	Last4         string `json:"last4"`
	AccountType   string `json:"account_type"`
	RoutingNumber string `json:"routing_number"`
	Country       string `json:"country"`
}

// AddPaymentMethodRequest represents add payment method request
type AddPaymentMethodRequest struct {
	Type       PaymentMethodType      `json:"type"`
	Token      string                 `json:"token"` // Payment provider token
	SetDefault bool                   `json:"set_default"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// UsageLimitStatus represents usage limit status
type UsageLimitStatus struct {
	TenantID     tenant.TenantID       `json:"tenant_id"`
	PlanLimits   subscription.Limits   `json:"plan_limits"`
	CurrentUsage subscription.Usage    `json:"current_usage"`
	Violations   []UsageLimitViolation `json:"violations"`
	Warnings     []UsageLimitWarning   `json:"warnings"`
	IsOverLimit  bool                  `json:"is_over_limit"`
}

// UsageLimitViolation represents a usage limit violation
type UsageLimitViolation struct {
	Resource   string            `json:"resource"`
	Limit      int64             `json:"limit"`
	Current    int64             `json:"current"`
	Percentage float64           `json:"percentage"`
	Severity   ViolationSeverity `json:"severity"`
}

// UsageLimitWarning represents a usage limit warning
type UsageLimitWarning struct {
	Resource   string  `json:"resource"`
	Limit      int64   `json:"limit"`
	Current    int64   `json:"current"`
	Percentage float64 `json:"percentage"`
	Threshold  float64 `json:"threshold"` // Warning threshold (e.g., 0.8 for 80%)
}

// ViolationSeverity represents violation severity
type ViolationSeverity string

const (
	SeverityWarning  ViolationSeverity = "warning"
	SeverityError    ViolationSeverity = "error"
	SeverityCritical ViolationSeverity = "critical"
)

// Provider-specific types

// CreateCustomerRequest represents customer creation request
type CreateCustomerRequest struct {
	TenantID tenant.TenantID        `json:"tenant_id"`
	Email    string                 `json:"email"`
	Name     string                 `json:"name"`
	Address  *Address               `json:"address"`
	Phone    *string                `json:"phone"`
	TaxInfo  *TaxInfo               `json:"tax_info"`
	Metadata map[string]interface{} `json:"metadata"`
}

// UpdateCustomerRequest represents customer update request
type UpdateCustomerRequest struct {
	Email    *string                `json:"email"`
	Name     *string                `json:"name"`
	Address  *Address               `json:"address"`
	Phone    *string                `json:"phone"`
	TaxInfo  *TaxInfo               `json:"tax_info"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Customer represents a customer in payment provider
type Customer struct {
	ID         string                 `json:"id"`
	TenantID   tenant.TenantID        `json:"tenant_id"`
	Email      string                 `json:"email"`
	Name       string                 `json:"name"`
	Address    *Address               `json:"address"`
	Phone      *string                `json:"phone"`
	TaxInfo    *TaxInfo               `json:"tax_info"`
	Currency   string                 `json:"currency"`
	Balance    shared.Money           `json:"balance"`
	Delinquent bool                   `json:"delinquent"`
	CreatedAt  shared.Timestamp       `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Address represents an address
type Address struct {
	Line1      string  `json:"line1"`
	Line2      *string `json:"line2"`
	City       string  `json:"city"`
	State      *string `json:"state"`
	PostalCode string  `json:"postal_code"`
	Country    string  `json:"country"`
}

// TaxInfo represents tax information
type TaxInfo struct {
	TaxID     *string `json:"tax_id"`
	TaxIDType *string `json:"tax_id_type"`
	TaxExempt bool    `json:"tax_exempt"`
}

// Additional provider types (Stripe, PayPal, etc.)
type ProviderSubscription struct {
	ID            string                     `json:"id"`
	CustomerID    string                     `json:"customer_id"`
	PlanID        string                     `json:"plan_id"`
	Status        ProviderSubscriptionStatus `json:"status"`
	CurrentPeriod BillingPeriod              `json:"current_period"`
	CancelAt      *shared.Timestamp          `json:"cancel_at"`
	CanceledAt    *shared.Timestamp          `json:"canceled_at"`
	TrialEnd      *shared.Timestamp          `json:"trial_end"`
	Metadata      map[string]interface{}     `json:"metadata"`
}

type ProviderSubscriptionStatus string

const (
	ProviderSubActive            ProviderSubscriptionStatus = "active"
	ProviderSubCanceled          ProviderSubscriptionStatus = "canceled"
	ProviderSubIncomplete        ProviderSubscriptionStatus = "incomplete"
	ProviderSubIncompleteExpired ProviderSubscriptionStatus = "incomplete_expired"
	ProviderSubPastDue           ProviderSubscriptionStatus = "past_due"
	ProviderSubTrialing          ProviderSubscriptionStatus = "trialing"
	ProviderSubUnpaid            ProviderSubscriptionStatus = "unpaid"
)

type CreateProviderSubscriptionRequest struct {
	CustomerID      string                 `json:"customer_id"`
	PlanID          string                 `json:"plan_id"`
	PaymentMethodID *string                `json:"payment_method_id"`
	TrialPeriodDays *int                   `json:"trial_period_days"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type UpdateProviderSubscriptionRequest struct {
	PlanID          *string                `json:"plan_id"`
	PaymentMethodID *string                `json:"payment_method_id"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type ProcessProviderPaymentRequest struct {
	CustomerID      string                 `json:"customer_id"`
	Amount          shared.Money           `json:"amount"`
	Currency        string                 `json:"currency"`
	PaymentMethodID string                 `json:"payment_method_id"`
	Description     string                 `json:"description"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type ProviderPayment struct {
	ID            string                 `json:"id"`
	CustomerID    string                 `json:"customer_id"`
	Amount        shared.Money           `json:"amount"`
	Currency      string                 `json:"currency"`
	Status        PaymentStatus          `json:"status"`
	PaymentMethod *ProviderPaymentMethod `json:"payment_method"`
	Description   string                 `json:"description"`
	CreatedAt     shared.Timestamp       `json:"created_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type CreatePaymentIntentRequest struct {
	CustomerID      string                 `json:"customer_id"`
	Amount          shared.Money           `json:"amount"`
	Currency        string                 `json:"currency"`
	PaymentMethodID *string                `json:"payment_method_id"`
	Description     string                 `json:"description"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type PaymentIntent struct {
	ID           string                 `json:"id"`
	CustomerID   string                 `json:"customer_id"`
	Amount       shared.Money           `json:"amount"`
	Currency     string                 `json:"currency"`
	Status       PaymentIntentStatus    `json:"status"`
	ClientSecret string                 `json:"client_secret"`
	CreatedAt    shared.Timestamp       `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type PaymentIntentStatus string

const (
	PaymentIntentRequiresPaymentMethod PaymentIntentStatus = "requires_payment_method"
	PaymentIntentRequiresConfirmation  PaymentIntentStatus = "requires_confirmation"
	PaymentIntentRequiresAction        PaymentIntentStatus = "requires_action"
	PaymentIntentProcessing            PaymentIntentStatus = "processing"
	PaymentIntentSucceeded             PaymentIntentStatus = "succeeded"
	PaymentIntentCanceled              PaymentIntentStatus = "canceled"
)

type Refund struct {
	ID        string           `json:"id"`
	PaymentID string           `json:"payment_id"`
	Amount    shared.Money     `json:"amount"`
	Currency  string           `json:"currency"`
	Status    RefundStatus     `json:"status"`
	Reason    string           `json:"reason"`
	CreatedAt shared.Timestamp `json:"created_at"`
}

type AddProviderPaymentMethodRequest struct {
	Type     PaymentMethodType      `json:"type"`
	Token    string                 `json:"token"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ProviderPaymentMethod struct {
	ID        string                 `json:"id"`
	Type      PaymentMethodType      `json:"type"`
	Card      *CardDetails           `json:"card,omitempty"`
	CreatedAt shared.Timestamp       `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type ProviderInvoiceFilter struct {
	Status    *InvoiceStatus
	DueAfter  *shared.Timestamp
	DueBefore *shared.Timestamp
	Limit     int
}

type ProviderInvoice struct {
	ID             string                 `json:"id"`
	CustomerID     string                 `json:"customer_id"`
	SubscriptionID *string                `json:"subscription_id"`
	Status         InvoiceStatus          `json:"status"`
	Amount         shared.Money           `json:"amount"`
	Currency       string                 `json:"currency"`
	Description    string                 `json:"description"`
	PeriodStart    *shared.Timestamp      `json:"period_start"`
	PeriodEnd      *shared.Timestamp      `json:"period_end"`
	DueDate        shared.Timestamp       `json:"due_date"`
	PaidAt         *shared.Timestamp      `json:"paid_at"`
	CreatedAt      shared.Timestamp       `json:"created_at"`
	HostedURL      string                 `json:"hosted_url"`
	DownloadURL    string                 `json:"download_url"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type BillingPortalSession struct {
	ID        string           `json:"id"`
	URL       string           `json:"url"`
	ReturnURL string           `json:"return_url"`
	ExpiresAt shared.Timestamp `json:"expires_at"`
	CreatedAt shared.Timestamp `json:"created_at"`
}

type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt shared.Timestamp       `json:"created_at"`
}

// Helper methods
func (s *UsageLimitStatus) HasViolations() bool {
	return len(s.Violations) > 0
}

func (s *UsageLimitStatus) HasWarnings() bool {
	return len(s.Warnings) > 0
}

func (s *UsageLimitStatus) GetViolationsByResource(resource string) []UsageLimitViolation {
	var violations []UsageLimitViolation
	for _, v := range s.Violations {
		if v.Resource == resource {
			violations = append(violations, v)
		}
	}
	return violations
}

func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}

func (i *Invoice) IsOverdue() bool {
	return i.Status == InvoiceStatusOpen && shared.Now().After(i.DueDate)
}
