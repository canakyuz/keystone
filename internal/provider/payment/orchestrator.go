package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/canakyuz/keystone/internal/config"
)

// Orchestrator manages multiple payment providers and routes requests
type Orchestrator struct {
	config    *config.PaymentConfig
	providers map[string]Provider
}

// NewOrchestrator creates a new payment orchestrator
func NewOrchestrator(cfg *config.PaymentConfig) *Orchestrator {
	return &Orchestrator{
		config:    cfg,
		providers: make(map[string]Provider),
	}
}

// RegisterProvider registers a payment provider
func (o *Orchestrator) RegisterProvider(name string, provider Provider) {
	o.providers[name] = provider
}

// GetProvider returns a provider by name
func (o *Orchestrator) GetProvider(name string) (Provider, error) {
	provider, exists := o.providers[name]
	if !exists {
		return nil, fmt.Errorf("payment provider %s not found", name)
	}
	return provider, nil
}

// SelectProvider selects the appropriate provider based on routing rules
// Priority: Tenant settings > Currency/Country > Default
func (o *Orchestrator) SelectProvider(tenantSettings map[string]interface{}, currency, country string) (Provider, error) {
	// 1. Check tenant-specific provider configuration
	if tenantProvider := o.getProviderFromTenantSettings(tenantSettings); tenantProvider != nil {
		return tenantProvider, nil
	}

	// 2. Route based on currency and country
	providerName := o.getProviderByCurrencyAndCountry(currency, country)
	if providerName != "" {
		return o.GetProvider(providerName)
	}

	// 3. Default provider (first available)
	for _, provider := range o.providers {
		return provider, nil
	}

	return nil, fmt.Errorf("no payment provider available")
}

// getProviderFromTenantSettings extracts provider from tenant settings
func (o *Orchestrator) getProviderFromTenantSettings(settings map[string]interface{}) Provider {
	if settings == nil {
		return nil
	}

	// Check if tenant has payment settings
	paymentSettings, ok := settings["payment"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Get preferred provider
	providerName, ok := paymentSettings["provider"].(string)
	if !ok || providerName == "" {
		return nil
	}

	// Get provider instance
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return nil
	}

	return provider
}

// getProviderByCurrencyAndCountry routes based on currency and country
func (o *Orchestrator) getProviderByCurrencyAndCountry(currency, country string) string {
	// Turkey-specific routing
	if currency == "TRY" || country == "TR" {
		if o.config.Iyzico.Enabled {
			return "iyzico"
		}
	}

	// European currencies
	if currency == "EUR" && o.config.Checkout.Enabled {
		return "checkout"
	}

	// US and other global currencies
	if (currency == "USD" || currency == "GBP") && o.config.Checkout.Enabled {
		return "checkout"
	}

	// Fallback to Stripe if enabled
	if o.config.Stripe.Enabled {
		return "stripe"
	}

	return ""
}

// CreatePayment creates a payment using the appropriate provider
func (o *Orchestrator) CreatePayment(ctx context.Context, tenantSettings map[string]interface{}, req *PaymentRequest) (*PaymentResponse, error) {
	provider, err := o.SelectProvider(tenantSettings, req.Currency, req.BillingCountry)
	if err != nil {
		return nil, fmt.Errorf("failed to select payment provider: %w", err)
	}

	return provider.CreatePayment(ctx, req)
}

// CompletePayment completes a payment (for 3DS)
func (o *Orchestrator) CompletePayment(ctx context.Context, providerName string, req *CompletePaymentRequest) (*PaymentResponse, error) {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.CompletePayment(ctx, req)
}

// GetPayment retrieves payment details
func (o *Orchestrator) GetPayment(ctx context.Context, providerName, providerPaymentID string) (*PaymentResponse, error) {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.GetPayment(ctx, providerPaymentID)
}

// CancelPayment cancels a payment
func (o *Orchestrator) CancelPayment(ctx context.Context, providerName, providerPaymentID string) error {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return err
	}

	return provider.CancelPayment(ctx, providerPaymentID)
}

// CreateRefund creates a refund
func (o *Orchestrator) CreateRefund(ctx context.Context, providerName string, req *RefundRequest) (*RefundResponse, error) {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.CreateRefund(ctx, req)
}

// GetRefund retrieves refund details
func (o *Orchestrator) GetRefund(ctx context.Context, providerName, providerRefundID string) (*RefundResponse, error) {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.GetRefund(ctx, providerRefundID)
}

// VerifyWebhookSignature verifies webhook signature
func (o *Orchestrator) VerifyWebhookSignature(ctx context.Context, providerName string, payload []byte, signature string) error {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return err
	}

	return provider.VerifyWebhookSignature(ctx, payload, signature)
}

// ParseWebhook parses webhook payload
func (o *Orchestrator) ParseWebhook(ctx context.Context, providerName string, payload []byte) (*WebhookEvent, error) {
	provider, err := o.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.ParseWebhook(ctx, payload)
}

// GetTenantProviderCredentials extracts provider credentials from tenant settings
func GetTenantProviderCredentials(settings map[string]interface{}, providerName string) (map[string]string, error) {
	if settings == nil {
		return nil, fmt.Errorf("tenant settings is nil")
	}

	// Get payment settings
	paymentSettings, ok := settings["payment"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payment settings not found in tenant settings")
	}

	// Get provider credentials
	providerSettings, ok := paymentSettings[providerName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s credentials not found in tenant settings", providerName)
	}

	// Convert to string map
	credentials := make(map[string]string)
	for key, value := range providerSettings {
		if strValue, ok := value.(string); ok {
			credentials[key] = strValue
		}
	}

	return credentials, nil
}

// ValidateTenantProviderSettings validates tenant provider configuration
func ValidateTenantProviderSettings(settings map[string]interface{}, providerName string) error {
	credentials, err := GetTenantProviderCredentials(settings, providerName)
	if err != nil {
		return err
	}

	// Validate based on provider
	switch providerName {
	case "iyzico":
		if credentials["api_key"] == "" || credentials["secret_key"] == "" {
			return fmt.Errorf("iyzico requires api_key and secret_key")
		}
	case "checkout":
		if credentials["public_key"] == "" || credentials["secret_key"] == "" {
			return fmt.Errorf("checkout.com requires public_key and secret_key")
		}
	case "stripe":
		if credentials["publishable_key"] == "" || credentials["secret_key"] == "" {
			return fmt.Errorf("stripe requires publishable_key and secret_key")
		}
	default:
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	return nil
}

// MarshalProviderSettings converts provider settings to JSON
func MarshalProviderSettings(providerName string, credentials map[string]string) ([]byte, error) {
	settings := map[string]interface{}{
		"payment": map[string]interface{}{
			"provider":   providerName,
			providerName: credentials,
		},
	}

	return json.Marshal(settings)
}
