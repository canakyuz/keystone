package external

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"nexspaces-api/internal/core/ports/services"
)

// Mock implementations for development and testing

// MockEmailService implements a mock email service
type MockEmailService struct{}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{}
}

func (m *MockEmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	log.Printf("MOCK EMAIL: Welcome email sent to %s (%s)", to, name)
	return nil
}

func (m *MockEmailService) SendInvitationEmail(ctx context.Context, to, inviterName, tenantName, inviteLink string) error {
	log.Printf("MOCK EMAIL: Invitation email sent to %s from %s for tenant %s", to, inviterName, tenantName)
	return nil
}

func (m *MockEmailService) SendPasswordResetEmail(ctx context.Context, to, resetLink string) error {
	log.Printf("MOCK EMAIL: Password reset email sent to %s", to)
	return nil
}

func (m *MockEmailService) SendSubscriptionNotification(ctx context.Context, to, subject, message string) error {
	log.Printf("MOCK EMAIL: Subscription notification sent to %s - %s", to, subject)
	return nil
}

// MockStorageService implements a mock storage service
type MockStorageService struct {
	files map[string][]byte
}

func NewMockStorageService() *MockStorageService {
	return &MockStorageService{
		files: make(map[string][]byte),
	}
}

func (m *MockStorageService) Upload(ctx context.Context, key string, content io.Reader, contentType string) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	m.files[key] = data
	mockURL := fmt.Sprintf("https://mock-storage.nexpaces.com/%s", key)
	log.Printf("MOCK STORAGE: File uploaded to %s (size: %d bytes, type: %s)", mockURL, len(data), contentType)
	return mockURL, nil
}

func (m *MockStorageService) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	data, exists := m.files[key]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", key)
	}
	log.Printf("MOCK STORAGE: File downloaded from %s (size: %d bytes)", key, len(data))
	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (m *MockStorageService) Delete(ctx context.Context, key string) error {
	delete(m.files, key)
	log.Printf("MOCK STORAGE: File deleted: %s", key)
	return nil
}

func (m *MockStorageService) GetURL(ctx context.Context, key string, expiry int) (string, error) {
	mockURL := fmt.Sprintf("https://mock-storage.nexpaces.com/%s?expires=%d", key, time.Now().Unix()+int64(expiry))
	log.Printf("MOCK STORAGE: Pre-signed URL generated for %s (expires in %d seconds)", key, expiry)
	return mockURL, nil
}

func (m *MockStorageService) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	for key := range m.files {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	log.Printf("MOCK STORAGE: Listed %d files with prefix %s", len(keys), prefix)
	return keys, nil
}

// MockBillingService implements a mock billing service
type MockBillingService struct {
	customers      map[string]*services.Customer
	subscriptions  map[string]*services.BillingSubscription
	invoices       map[string][]*services.Invoice
	paymentMethods map[string]*services.PaymentMethod
}

func NewMockBillingService() *MockBillingService {
	return &MockBillingService{
		customers:      make(map[string]*services.Customer),
		subscriptions:  make(map[string]*services.BillingSubscription),
		invoices:       make(map[string][]*services.Invoice),
		paymentMethods: make(map[string]*services.PaymentMethod),
	}
}

func (m *MockBillingService) CreateCustomer(ctx context.Context, req services.CreateCustomerRequest) (*services.Customer, error) {
	customer := &services.Customer{
		ID:       req.TenantID,
		TenantID: req.TenantID,
		Email:    req.Email,
		Name:     req.Name,
		StripeID: fmt.Sprintf("mock_cus_%s", req.TenantID),
	}
	m.customers[req.TenantID] = customer
	log.Printf("MOCK BILLING: Customer created for tenant %s (%s)", req.TenantID, req.Email)
	return customer, nil
}

func (m *MockBillingService) UpdateCustomer(ctx context.Context, customerID string, req services.UpdateCustomerRequest) (*services.Customer, error) {
	customer, exists := m.customers[customerID]
	if !exists {
		return nil, fmt.Errorf("customer not found: %s", customerID)
	}

	if req.Email != nil {
		customer.Email = *req.Email
	}
	if req.Name != nil {
		customer.Name = *req.Name
	}

	log.Printf("MOCK BILLING: Customer updated: %s", customerID)
	return customer, nil
}

func (m *MockBillingService) CreateSubscription(ctx context.Context, req services.CreateSubscriptionRequest) (*services.BillingSubscription, error) {
	subscriptionID := fmt.Sprintf("mock_sub_%s_%d", req.CustomerID, time.Now().Unix())

	subscription := &services.BillingSubscription{
		ID:           subscriptionID,
		CustomerID:   req.CustomerID,
		PlanID:       req.PlanID,
		Status:       "active",
		BillingCycle: req.BillingCycle,
		Amount:       29.99, // Mock price
		Currency:     "usd",
		StripeID:     subscriptionID,
	}

	if req.TrialDays != nil && *req.TrialDays > 0 {
		subscription.Status = "trialing"
	}

	m.subscriptions[subscriptionID] = subscription
	log.Printf("MOCK BILLING: Subscription created: %s for customer %s", subscriptionID, req.CustomerID)
	return subscription, nil
}

func (m *MockBillingService) UpdateSubscription(ctx context.Context, subscriptionID string, req services.UpdateSubscriptionRequest) (*services.BillingSubscription, error) {
	subscription, exists := m.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	if req.PlanID != nil {
		subscription.PlanID = *req.PlanID
	}
	if req.BillingCycle != nil {
		subscription.BillingCycle = *req.BillingCycle
	}

	log.Printf("MOCK BILLING: Subscription updated: %s", subscriptionID)
	return subscription, nil
}

func (m *MockBillingService) CancelSubscription(ctx context.Context, subscriptionID string) error {
	subscription, exists := m.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	subscription.Status = "canceled"
	log.Printf("MOCK BILLING: Subscription canceled: %s", subscriptionID)
	return nil
}

func (m *MockBillingService) GetSubscription(ctx context.Context, subscriptionID string) (*services.BillingSubscription, error) {
	subscription, exists := m.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	log.Printf("MOCK BILLING: Subscription retrieved: %s", subscriptionID)
	return subscription, nil
}

func (m *MockBillingService) CreatePaymentMethod(ctx context.Context, customerID string, req services.CreatePaymentMethodRequest) (*services.PaymentMethod, error) {
	paymentMethodID := fmt.Sprintf("mock_pm_%s_%d", customerID, time.Now().Unix())

	paymentMethod := &services.PaymentMethod{
		ID:         paymentMethodID,
		CustomerID: customerID,
		Type:       req.Type,
		Last4:      "4242",
		Brand:      "visa",
		IsDefault:  true,
		StripeID:   paymentMethodID,
	}

	m.paymentMethods[paymentMethodID] = paymentMethod
	log.Printf("MOCK BILLING: Payment method created: %s for customer %s", paymentMethodID, customerID)
	return paymentMethod, nil
}

func (m *MockBillingService) GetInvoices(ctx context.Context, customerID string, limit int) ([]*services.Invoice, error) {
	invoices, exists := m.invoices[customerID]
	if !exists {
		// Generate mock invoices
		invoices = []*services.Invoice{
			{
				ID:          fmt.Sprintf("mock_in_%s_1", customerID),
				CustomerID:  customerID,
				Amount:      29.99,
				Currency:    "usd",
				Status:      "paid",
				InvoiceDate: time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
				DueDate:     time.Now().AddDate(0, -1, 15).Format(time.RFC3339),
				InvoiceURL:  fmt.Sprintf("https://mock-billing.nexpaces.com/invoices/mock_in_%s_1.pdf", customerID),
				StripeID:    fmt.Sprintf("mock_in_%s_1", customerID),
			},
		}
		m.invoices[customerID] = invoices
	}

	if len(invoices) > limit {
		invoices = invoices[:limit]
	}

	log.Printf("MOCK BILLING: Retrieved %d invoices for customer %s", len(invoices), customerID)
	return invoices, nil
}

func (m *MockBillingService) ProcessWebhook(ctx context.Context, payload []byte, signature string) error {
	log.Printf("MOCK BILLING: Webhook processed (payload size: %d bytes)", len(payload))
	return nil
}

// GetPlan retrieves plan details (dummy implementation)
func (m *MockBillingService) GetPlan(ctx context.Context, planID services.PlanID) (*services.Plan, error) {
	log.Printf("MOCK BILLING: GetPlan called for plan %s", planID)
	return &services.Plan{
		ID:       planID,
		Price:    10.00,
		Currency: "usd",
		Features: []string{"feature1", "feature2"},
		Limits:   map[string]int64{"requests": 1000},
	}, nil
}

// ProcessSubscriptionPayment processes a subscription payment (dummy implementation)
func (m *MockBillingService) ProcessSubscriptionPayment(ctx context.Context, req services.SubscriptionPaymentRequest) (*services.SubscriptionPaymentResult, error) {
	log.Printf("MOCK BILLING: ProcessSubscriptionPayment called for subscription %s", req.SubscriptionID)
	return &services.SubscriptionPaymentResult{
		SubscriptionID:  string(req.SubscriptionID),
		PaymentIntentID: "pi_mock_" + string(req.SubscriptionID),
		Status:          "active",
	}, nil
}
