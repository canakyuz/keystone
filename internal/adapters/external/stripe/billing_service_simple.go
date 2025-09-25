package stripe

import (
	"context"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/ports/services"
)

// SimpleBillingService implements the billing service using Stripe with a simplified approach
type SimpleBillingService struct {
	client *client.API
}

// NewSimpleBillingService creates a new Stripe billing service
func NewSimpleBillingService(secretKey string) *SimpleBillingService {
	stripeClient := &client.API{}
	stripeClient.Init(secretKey, nil)

	return &SimpleBillingService{
		client: stripeClient,
	}
}

// CreateCustomer creates a new Stripe customer
func (s *SimpleBillingService) CreateCustomer(ctx context.Context, req services.CreateCustomerRequest) (*services.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
		Name:  stripe.String(req.Name),
		Metadata: map[string]string{
			"tenant_id": req.TenantID,
		},
	}

	customer, err := s.client.Customers.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return &services.Customer{
		ID:       customer.Metadata["tenant_id"], // Use tenant_id as our internal ID
		TenantID: req.TenantID,
		Email:    customer.Email,
		Name:     customer.Name,
		StripeID: customer.ID,
	}, nil
}

// UpdateCustomer updates an existing Stripe customer
func (s *SimpleBillingService) UpdateCustomer(ctx context.Context, customerID string, req services.UpdateCustomerRequest) (*services.Customer, error) {
	// In a real implementation, you'd store the Stripe customer ID in your database
	// For this example, we assume customerID is the Stripe customer ID
	params := &stripe.CustomerParams{}
	if req.Email != nil {
		params.Email = stripe.String(*req.Email)
	}
	if req.Name != nil {
		params.Name = stripe.String(*req.Name)
	}

	updatedCustomer, err := s.client.Customers.Update(customerID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe customer: %w", err)
	}

	return &services.Customer{
		ID:       updatedCustomer.Metadata["tenant_id"],
		TenantID: updatedCustomer.Metadata["tenant_id"],
		Email:    updatedCustomer.Email,
		Name:     updatedCustomer.Name,
		StripeID: updatedCustomer.ID,
	}, nil
}

// CreateSubscription creates a new Stripe subscription
func (s *SimpleBillingService) CreateSubscription(ctx context.Context, req services.CreateSubscriptionRequest) (*services.BillingSubscription, error) {
	// In a real implementation, req.CustomerID would be the Stripe customer ID from your database
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(req.CustomerID), // Assuming this is the Stripe customer ID
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(req.PlanID), // Assuming PlanID is the Stripe Price ID
			},
		},
		Metadata: map[string]string{
			"billing_cycle": req.BillingCycle,
		},
	}

	// Add trial period if specified
	if req.TrialDays != nil && *req.TrialDays > 0 {
		params.TrialPeriodDays = stripe.Int64(int64(*req.TrialDays))
	}

	subscription, err := s.client.Subscriptions.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	amount := float64(0)
	if len(subscription.Items.Data) > 0 {
		amount = float64(subscription.Items.Data[0].Price.UnitAmount) / 100 // Convert from cents
	}

	return &services.BillingSubscription{
		ID:           subscription.ID, // Use Stripe subscription ID
		CustomerID:   req.CustomerID,
		PlanID:       req.PlanID,
		Status:       string(subscription.Status),
		BillingCycle: req.BillingCycle,
		Amount:       amount,
		Currency:     string(subscription.Currency),
		StripeID:     subscription.ID,
	}, nil
}

// UpdateSubscription updates an existing Stripe subscription
func (s *SimpleBillingService) UpdateSubscription(ctx context.Context, subscriptionID string, req services.UpdateSubscriptionRequest) (*services.BillingSubscription, error) {
	params := &stripe.SubscriptionParams{}

	// Get current subscription first
	currentSubscription, err := s.client.Subscriptions.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current subscription: %w", err)
	}

	if req.PlanID != nil {
		// Update subscription items
		params.Items = []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(currentSubscription.Items.Data[0].ID),
				Price: stripe.String(*req.PlanID),
			},
		}
	}

	if req.BillingCycle != nil {
		if params.Metadata == nil {
			params.Metadata = make(map[string]string)
		}
		params.Metadata["billing_cycle"] = *req.BillingCycle
	}

	updatedSubscription, err := s.client.Subscriptions.Update(subscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	amount := float64(0)
	if len(updatedSubscription.Items.Data) > 0 {
		amount = float64(updatedSubscription.Items.Data[0].Price.UnitAmount) / 100
	}

	planID := ""
	if req.PlanID != nil {
		planID = *req.PlanID
	} else if len(updatedSubscription.Items.Data) > 0 {
		planID = updatedSubscription.Items.Data[0].Price.ID
	}

	return &services.BillingSubscription{
		ID:           updatedSubscription.ID,
		CustomerID:   updatedSubscription.Customer.ID,
		PlanID:       planID,
		Status:       string(updatedSubscription.Status),
		BillingCycle: updatedSubscription.Metadata["billing_cycle"],
		Amount:       amount,
		Currency:     string(updatedSubscription.Currency),
		StripeID:     updatedSubscription.ID,
	}, nil
}

// CancelSubscription cancels a Stripe subscription
func (s *SimpleBillingService) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, err := s.client.Subscriptions.Cancel(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	return nil
}

// GetSubscription retrieves a Stripe subscription
func (s *SimpleBillingService) GetSubscription(ctx context.Context, subscriptionID string) (*services.BillingSubscription, error) {
	subscription, err := s.client.Subscriptions.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe subscription: %w", err)
	}

	amount := float64(0)
	planID := ""
	if len(subscription.Items.Data) > 0 {
		amount = float64(subscription.Items.Data[0].Price.UnitAmount) / 100
		planID = subscription.Items.Data[0].Price.ID
	}

	return &services.BillingSubscription{
		ID:           subscription.ID,
		CustomerID:   subscription.Customer.ID,
		PlanID:       planID,
		Status:       string(subscription.Status),
		BillingCycle: subscription.Metadata["billing_cycle"],
		Amount:       amount,
		Currency:     string(subscription.Currency),
		StripeID:     subscription.ID,
	}, nil
}

// CreatePaymentMethod creates a payment method (simplified)
func (s *SimpleBillingService) CreatePaymentMethod(ctx context.Context, customerID string, req services.CreatePaymentMethodRequest) (*services.PaymentMethod, error) {
	// This is a simplified implementation
	// In a real-world scenario, you'd handle payment method creation differently
	// typically involving frontend integration with Stripe Elements

	return &services.PaymentMethod{
		ID:         "pm_placeholder",
		CustomerID: customerID,
		Type:       req.Type,
		IsDefault:  true,
		StripeID:   "pm_placeholder",
	}, nil
}

// GetInvoices retrieves invoices for a customer
func (s *SimpleBillingService) GetInvoices(ctx context.Context, customerID string, limit int) ([]*services.Invoice, error) {
	params := &stripe.InvoiceListParams{
		Customer: stripe.String(customerID), // Assuming customerID is Stripe customer ID
	}
	params.Limit = stripe.Int64(int64(limit))

	invoiceList := s.client.Invoices.List(params)

	var invoices []*services.Invoice
	for invoiceList.Next() {
		invoice := invoiceList.Invoice()

		// Convert Unix timestamp to string
		invoiceDate := time.Unix(invoice.Created, 0).Format(time.RFC3339)
		dueDate := time.Unix(invoice.DueDate, 0).Format(time.RFC3339)

		invoices = append(invoices, &services.Invoice{
			ID:          invoice.ID,
			CustomerID:  customerID,
			Amount:      float64(invoice.AmountPaid) / 100,
			Currency:    string(invoice.Currency),
			Status:      string(invoice.Status),
			InvoiceDate: invoiceDate,
			DueDate:     dueDate,
			InvoiceURL:  invoice.InvoicePDF,
			StripeID:    invoice.ID,
		})
	}

	return invoices, nil
}

// ProcessWebhook processes Stripe webhook events
func (s *SimpleBillingService) ProcessWebhook(ctx context.Context, payload []byte, signature string) error {
	// Webhook processing would go here
	// This is a placeholder implementation
	// In production, you'd parse the webhook event and handle different event types

	// Example event types to handle:
	// - customer.created
	// - customer.updated
	// - invoice.payment_succeeded
	// - invoice.payment_failed
	// - customer.subscription.created
	// - customer.subscription.updated
	// - customer.subscription.deleted

	return nil
}

// GetPlan retrieves plan details (dummy implementation)
func (s *SimpleBillingService) GetPlan(ctx context.Context, planID shared.PlanID) (*services.Plan, error) {
	// This is a dummy implementation. In a real application, you would fetch plan details from Stripe.
	return &services.Plan{
		ID:       planID,
		Price:    10.00,
		Currency: "usd",
		Features: []string{"feature1", "feature2"},
		Limits:   map[string]int64{"requests": 1000},
	}, nil
}

// ProcessSubscriptionPayment processes a subscription payment (dummy implementation)
func (s *SimpleBillingService) ProcessSubscriptionPayment(ctx context.Context, req services.SubscriptionPaymentRequest) (*services.SubscriptionPaymentResult, error) {
	// This is a dummy implementation. In a real application, you would process the payment with Stripe.
	return &services.SubscriptionPaymentResult{
		SubscriptionID:  "sub_placeholder",
		PaymentIntentID: "pi_placeholder",
		Status:          "active",
	}, nil
}
