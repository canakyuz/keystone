package services

import (
	"context"
	"io"

	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/shared"
)

// AuthService represents the authentication service interface
type AuthService interface {
	// GenerateInviteToken creates a new token for a user to complete registration
	GenerateInviteToken(ctx context.Context, userID uuid.UUID, tenantID shared.TenantID) (string, error)
}

// EmailService represents the email service interface
type EmailService interface {
	// SendWelcomeEmail sends a welcome email to a new user
	SendWelcomeEmail(ctx context.Context, to, name string) error

	// SendInvitationEmail sends an invitation email
	SendInvitationEmail(ctx context.Context, to, inviterName, tenantName, inviteLink string) error

	// SendPasswordResetEmail sends a password reset email
	SendPasswordResetEmail(ctx context.Context, to, resetLink string) error

	// SendSubscriptionNotification sends subscription-related notifications
	SendSubscriptionNotification(ctx context.Context, to, subject, message string) error
}

// StorageService represents the file storage service interface
type StorageService interface {
	// Upload uploads a file and returns the URL
	Upload(ctx context.Context, key string, content io.Reader, contentType string) (string, error)

	// Download downloads a file
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete deletes a file
	Delete(ctx context.Context, key string) error

	// GetURL gets a pre-signed URL for a file
	GetURL(ctx context.Context, key string, expiry int) (string, error)

	// List lists files with a prefix
	List(ctx context.Context, prefix string) ([]string, error)
}

// BillingService represents the billing service interface (Stripe)
type BillingService interface {
	// CreateCustomer creates a new customer
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*Customer, error)

	// UpdateCustomer updates customer information
	UpdateCustomer(ctx context.Context, customerID string, req UpdateCustomerRequest) (*Customer, error)

	// CreateSubscription creates a new subscription
	CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*BillingSubscription, error)

	// UpdateSubscription updates a subscription
	UpdateSubscription(ctx context.Context, subscriptionID string, req UpdateSubscriptionRequest) (*BillingSubscription, error)

	// CancelSubscription cancels a subscription
	CancelSubscription(ctx context.Context, subscriptionID string) error

	// GetSubscription retrieves subscription details
	GetSubscription(ctx context.Context, subscriptionID string) (*BillingSubscription, error)

	// CreatePaymentMethod creates a payment method
	CreatePaymentMethod(ctx context.Context, customerID string, req CreatePaymentMethodRequest) (*PaymentMethod, error)

	// GetInvoices retrieves invoices for a customer
	GetInvoices(ctx context.Context, customerID string, limit int) ([]*Invoice, error)

	// ProcessWebhook processes webhook events
	ProcessWebhook(ctx context.Context, payload []byte, signature string) error

	// GetPlan retrieves plan details
	GetPlan(ctx context.Context, planID shared.PlanID) (*Plan, error)

	// ProcessSubscriptionPayment processes a subscription payment
	ProcessSubscriptionPayment(ctx context.Context, req SubscriptionPaymentRequest) (*SubscriptionPaymentResult, error)
}

// NotificationService represents the notification service interface
type NotificationService interface {
	// SendNotification sends a notification to a user
	SendNotification(ctx context.Context, userID, tenantID, title, message string, notificationType NotificationType) error

	// GetNotifications retrieves notifications for a user
	GetNotifications(ctx context.Context, userID, tenantID string, limit, offset int) ([]*Notification, error)

	// MarkAsRead marks notifications as read
	MarkAsRead(ctx context.Context, notificationIDs []string) error

	// DeleteNotification deletes a notification
	DeleteNotification(ctx context.Context, notificationID string) error

	// SendUserInvitation sends an invitation email to a new user
	SendUserInvitation(ctx context.Context, req UserInvitationRequest) error
}

// Billing Service DTOs

type Plan struct {
	ID       shared.PlanID    `json:"id"`
	Price    float64          `json:"price"`
	Currency string           `json:"currency"`
	Features []string         `json:"features"`
	Limits   map[string]int64 `json:"limits"`
}

type SubscriptionPaymentRequest struct {
	TenantID        shared.TenantID       `json:"tenant_id"`
	SubscriptionID  shared.SubscriptionID `json:"subscription_id"`
	PlanID          shared.PlanID         `json:"plan_id"`
	Amount          float64               `json:"amount"`
	Currency        string                `json:"currency"`
	BillingCycle    string                `json:"billing_cycle"`
	PaymentMethodID string                `json:"payment_method_id"`
}

type SubscriptionPaymentResult struct {
	SubscriptionID  string `json:"subscription_id"`
	PaymentIntentID string `json:"payment_intent_id"`
	Status          string `json:"status"`
}

type UserInvitationRequest struct {
	UserID      uuid.UUID       `json:"user_id"`
	TenantID    shared.TenantID `json:"tenant_id"`
	Email       string          `json:"email"`
	InviteToken string          `json:"invite_token"`
	InvitedBy   string          `json:"invited_by"`
}

// Billing Service DTOs

type CreateCustomerRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type UpdateCustomerRequest struct {
	Email *string `json:"email,omitempty"`
	Name  *string `json:"name,omitempty"`
}

type Customer struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	StripeID string `json:"stripe_id"`
}

type CreateSubscriptionRequest struct {
	CustomerID   string `json:"customer_id"`
	PlanID       string `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"`
	TrialDays    *int   `json:"trial_days,omitempty"`
}

type UpdateSubscriptionRequest struct {
	PlanID       *string `json:"plan_id,omitempty"`
	BillingCycle *string `json:"billing_cycle,omitempty"`
}

type BillingSubscription struct {
	ID           string  `json:"id"`
	CustomerID   string  `json:"customer_id"`
	PlanID       string  `json:"plan_id"`
	Status       string  `json:"status"`
	BillingCycle string  `json:"billing_cycle"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	StripeID     string  `json:"stripe_id"`
}

type CreatePaymentMethodRequest struct {
	Type     string            `json:"type"`
	CardData map[string]string `json:"card_data,omitempty"`
}

type PaymentMethod struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	Type        string `json:"type"`
	Last4       string `json:"last4,omitempty"`
	Brand       string `json:"brand,omitempty"`
	ExpiryMonth int    `json:"expiry_month,omitempty"`
	ExpiryYear  int    `json:"expiry_year,omitempty"`
	IsDefault   bool   `json:"is_default"`
	StripeID    string `json:"stripe_id"`
}

type Invoice struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	InvoiceDate string  `json:"invoice_date"`
	DueDate     string  `json:"due_date"`
	InvoiceURL  string  `json:"invoice_url"`
	StripeID    string  `json:"stripe_id"`
}

// Notification Service DTOs

type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
)

type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	TenantID  string           `json:"tenant_id"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Type      NotificationType `json:"type"`
	IsRead    bool             `json:"is_read"`
	CreatedAt string           `json:"created_at"`
}

// AnalyticsService represents the analytics service interface
type AnalyticsService interface {
	// TrackEvent tracks an analytics event
	TrackEvent(ctx context.Context, tenantID, userID, eventName string, properties map[string]interface{}) error

	// GetAnalytics retrieves analytics data
	GetAnalytics(ctx context.Context, tenantID string, req AnalyticsRequest) (*AnalyticsResponse, error)

	// GetUsageMetrics retrieves usage metrics for a tenant
	GetUsageMetrics(ctx context.Context, tenantID string, period string) (*UsageMetrics, error)
}

type AnalyticsRequest struct {
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	Metrics   []string `json:"metrics"`
	GroupBy   string   `json:"group_by,omitempty"`
}

type AnalyticsResponse struct {
	Data   []map[string]interface{} `json:"data"`
	Total  int                      `json:"total"`
	Period string                   `json:"period"`
}

type UsageMetrics struct {
	TenantID    string `json:"tenant_id"`
	Period      string `json:"period"`
	Templates   int    `json:"templates"`
	Users       int    `json:"users"`
	APIRequests int    `json:"api_requests"`
	StorageUsed int64  `json:"storage_used_bytes"`
	PageViews   int    `json:"page_views"`
	ActiveUsers int    `json:"active_users"`
}
