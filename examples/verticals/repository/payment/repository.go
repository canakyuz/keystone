package payment

import (
	"context"

	"github.com/canakyuz/keystone/examples/verticals/domain/payment"
	"github.com/canakyuz/keystone/examples/verticals/domain/refund"
	"github.com/canakyuz/keystone/examples/verticals/domain/webhook"
)

// Repository defines payment data access interface
type Repository interface {
	// Payment operations
	CreatePayment(ctx context.Context, payment *payment.Payment) error
	GetPaymentByID(ctx context.Context, id string) (*payment.Payment, error)
	GetPaymentByProviderID(ctx context.Context, tenantID, providerPaymentID string) (*payment.Payment, error)
	UpdatePayment(ctx context.Context, payment *payment.Payment) error
	ListPaymentsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*payment.Payment, error)

	// Refund operations
	CreateRefund(ctx context.Context, refund *refund.Refund) error
	GetRefundByID(ctx context.Context, id string) (*refund.Refund, error)
	GetRefundByProviderID(ctx context.Context, tenantID, providerRefundID string) (*refund.Refund, error)
	UpdateRefund(ctx context.Context, refund *refund.Refund) error
	ListRefundsByPayment(ctx context.Context, paymentID string) ([]*refund.Refund, error)
	ListRefundsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*refund.Refund, error)

	// Webhook event operations
	CreateEvent(ctx context.Context, event *webhook.PaymentEvent) error
	GetEventByID(ctx context.Context, id string) (*webhook.PaymentEvent, error)
	GetEventByProviderEventID(ctx context.Context, tenantID, provider, providerEventID string) (*webhook.PaymentEvent, error)
	UpdateEvent(ctx context.Context, event *webhook.PaymentEvent) error
	ListUnprocessedEvents(ctx context.Context, limit int) ([]*webhook.PaymentEvent, error)
	ListEventsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*webhook.PaymentEvent, error)
}
