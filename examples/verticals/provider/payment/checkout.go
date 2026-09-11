package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/canakyuz/keystone/internal/config"
)

// CheckoutProvider implements Checkout.com payment integration for global markets
type CheckoutProvider struct {
	config *config.CheckoutConfig
	client *http.Client
}

// NewCheckoutProvider creates a new Checkout.com provider
func NewCheckoutProvider(cfg *config.CheckoutConfig) *CheckoutProvider {
	return &CheckoutProvider{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName returns the provider name
func (p *CheckoutProvider) GetName() string {
	return "checkout"
}

// CreatePayment initiates a payment with Checkout.com
func (p *CheckoutProvider) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	// Build Checkout.com payment request
	checkoutReq := map[string]any{
		"source": map[string]any{
			"type":  "token",
			"token": req.CardToken, // Checkout.com uses tokenized cards
		},
		"amount":      int(req.Amount * 100), // Amount in cents
		"currency":    req.Currency,
		"reference":   req.ReferenceID,
		"description": req.Description,

		// 3DS configuration
		"3ds": map[string]any{
			"enabled": true,
		},

		// Customer information
		"customer": map[string]any{
			"email": req.CustomerEmail,
			"name":  req.CustomerFirstName + " " + req.CustomerLastName,
		},

		// Billing address
		"billing": map[string]any{
			"address": map[string]any{
				"address_line1": req.BillingAddressLine,
				"city":          req.BillingCity,
				"zip":           req.BillingZipCode,
				"country":       req.BillingCountry,
			},
		},

		// Success and failure URLs
		"success_url": req.SuccessURL,
		"failure_url": req.FailureURL,

		// Metadata
		"metadata": map[string]any{
			"tenant_id": req.TenantID,
			"order_id":  req.OrderID,
		},
	}

	// Make API request
	respData, err := p.makeRequest(ctx, "/payments", "POST", checkoutReq)
	if err != nil {
		return nil, fmt.Errorf("checkout.com API error: %w", err)
	}

	// Parse response
	paymentID, _ := respData["id"].(string)
	status, _ := respData["status"].(string)
	approved, _ := respData["approved"].(bool)

	// Check for 3DS redirect
	var requires3DS bool
	var threeDSRedirectURL string

	if links, ok := respData["_links"].(map[string]any); ok {
		if redirectLink, ok := links["redirect"].(map[string]any); ok {
			if href, ok := redirectLink["href"].(string); ok {
				requires3DS = true
				threeDSRedirectURL = href
			}
		}
	}

	// Extract card details
	var cardBrand, cardLast4, cardBin, cardExpMonth, cardExpYear string
	if source, ok := respData["source"].(map[string]any); ok {
		cardBrand, _ = source["scheme"].(string)
		cardLast4, _ = source["last4"].(string)
		cardBin, _ = source["bin"].(string)
		if expMonth, ok := source["expiry_month"].(float64); ok {
			cardExpMonth = fmt.Sprintf("%02d", int(expMonth))
		}
		if expYear, ok := source["expiry_year"].(float64); ok {
			cardExpYear = fmt.Sprintf("%d", int(expYear))
		}
	}

	// Determine payment status
	paymentStatus := "pending"
	if approved {
		paymentStatus = "succeeded"
	} else if requires3DS {
		paymentStatus = "requires_3ds"
	} else if status == "Declined" {
		paymentStatus = "failed"
	}

	// Extract failure details if any
	var failureCode, failureMessage string
	if responseCode, ok := respData["response_code"].(string); ok {
		failureCode = responseCode
	}
	if responseSummary, ok := respData["response_summary"].(string); ok {
		failureMessage = responseSummary
	}

	response := &PaymentResponse{
		ProviderPaymentID:  paymentID,
		Status:             paymentStatus,
		Amount:             req.Amount,
		Currency:           req.Currency,
		Requires3DS:        requires3DS,
		ThreeDSRedirectURL: threeDSRedirectURL,
		CardBrand:          mapCheckoutCardBrand(cardBrand),
		CardLast4:          cardLast4,
		CardBIN:            cardBin,
		CardExpMonth:       cardExpMonth,
		CardExpYear:        cardExpYear,
		FailureCode:        failureCode,
		FailureMessage:     failureMessage,
		RawResponse:        respData,
	}

	return response, nil
}

// CompletePayment completes 3DS authentication (Checkout.com handles this automatically via redirect)
func (p *CheckoutProvider) CompletePayment(ctx context.Context, req *CompletePaymentRequest) (*PaymentResponse, error) {
	// Get payment details to check if 3DS completed
	return p.GetPayment(ctx, req.ProviderPaymentID)
}

// GetPayment retrieves payment details
func (p *CheckoutProvider) GetPayment(ctx context.Context, providerPaymentID string) (*PaymentResponse, error) {
	// Make API request
	endpoint := fmt.Sprintf("/payments/%s", providerPaymentID)
	respData, err := p.makeRequest(ctx, endpoint, "GET", nil)
	if err != nil {
		return nil, fmt.Errorf("checkout.com get payment error: %w", err)
	}

	// Parse response
	status, _ := respData["status"].(string)
	approved, _ := respData["approved"].(bool)
	amountFloat, _ := respData["amount"].(float64)
	amount := amountFloat / 100 // Convert from cents
	currency, _ := respData["currency"].(string)

	// Determine status
	paymentStatus := mapCheckoutStatus(status, approved)

	// Extract card details
	var cardBrand, cardLast4 string
	if source, ok := respData["source"].(map[string]any); ok {
		cardBrand, _ = source["scheme"].(string)
		cardLast4, _ = source["last4"].(string)
	}

	return &PaymentResponse{
		ProviderPaymentID: providerPaymentID,
		Status:            paymentStatus,
		Amount:            amount,
		Currency:          currency,
		CardBrand:         mapCheckoutCardBrand(cardBrand),
		CardLast4:         cardLast4,
		RawResponse:       respData,
	}, nil
}

// CancelPayment cancels a payment (Checkout.com uses void)
func (p *CheckoutProvider) CancelPayment(ctx context.Context, providerPaymentID string) error {
	// Build request
	voidReq := map[string]any{
		"reference": fmt.Sprintf("void_%s", providerPaymentID),
	}

	// Make API request
	endpoint := fmt.Sprintf("/payments/%s/voids", providerPaymentID)
	respData, err := p.makeRequest(ctx, endpoint, "POST", voidReq)
	if err != nil {
		return fmt.Errorf("checkout.com void error: %w", err)
	}

	// Check response
	if actionID, ok := respData["action_id"].(string); !ok || actionID == "" {
		return fmt.Errorf("void failed: no action ID returned")
	}

	return nil
}

// CreateRefund initiates a refund
func (p *CheckoutProvider) CreateRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// Build request
	refundReq := map[string]any{
		"amount":    int(req.Amount * 100), // Amount in cents
		"reference": req.PaymentID,
		"metadata": map[string]any{
			"tenant_id":   req.TenantID,
			"reason":      req.Reason,
			"description": req.Description,
		},
	}

	// Make API request
	endpoint := fmt.Sprintf("/payments/%s/refunds", req.ProviderPaymentID)
	respData, err := p.makeRequest(ctx, endpoint, "POST", refundReq)
	if err != nil {
		return nil, fmt.Errorf("checkout.com refund error: %w", err)
	}

	// Parse response
	actionID, _ := respData["action_id"].(string)

	return &RefundResponse{
		ProviderRefundID: actionID,
		Status:           "succeeded",
		Amount:           req.Amount,
		Currency:         req.Currency,
		RawResponse:      respData,
	}, nil
}

// GetRefund retrieves refund details
func (p *CheckoutProvider) GetRefund(ctx context.Context, providerRefundID string) (*RefundResponse, error) {
	// Checkout.com refunds are actions, retrieve via payment actions
	// This would require payment ID which we don't have here
	return nil, fmt.Errorf("checkout.com refund retrieval requires payment ID")
}

// VerifyWebhookSignature verifies webhook signature
func (p *CheckoutProvider) VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error {
	// Generate expected signature
	expectedSignature := p.generateWebhookSignature(payload)

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid webhook signature")
	}

	return nil
}

// ParseWebhook parses webhook payload
func (p *CheckoutProvider) ParseWebhook(ctx context.Context, payload []byte) (*WebhookEvent, error) {
	var webhookData map[string]any
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Extract event details
	eventType, _ := webhookData["type"].(string)
	eventID, _ := webhookData["id"].(string)

	// Extract payment data
	var paymentID, status string
	var amount float64
	var currency string

	if data, ok := webhookData["data"].(map[string]any); ok {
		paymentID, _ = data["id"].(string)
		status, _ = data["status"].(string)
		if amountFloat, ok := data["amount"].(float64); ok {
			amount = amountFloat / 100 // Convert from cents
		}
		currency, _ = data["currency"].(string)
	}

	event := &WebhookEvent{
		ProviderEventID: eventID,
		EventType:       mapCheckoutWebhookEvent(eventType),
		EventVersion:    "1.0",
		PaymentID:       paymentID,
		Status:          mapCheckoutWebhookStatus(status),
		Amount:          amount,
		Currency:        currency,
		Payload:         webhookData,
	}

	return event, nil
}

// makeRequest makes HTTP request to Checkout.com API
func (p *CheckoutProvider) makeRequest(ctx context.Context, endpoint, method string, body any) (map[string]any, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Create request
	url := p.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", p.config.SecretKey)

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return result, nil
}

// generateWebhookSignature generates HMAC signature for webhook verification
func (p *CheckoutProvider) generateWebhookSignature(payload []byte) string {
	// Checkout.com uses HMAC-SHA256
	hash := hmac.New(sha256.New, []byte(p.config.SecretKey))
	hash.Write(payload)
	return hex.EncodeToString(hash.Sum(nil))
}

// mapCheckoutStatus maps Checkout.com status to standard status
func mapCheckoutStatus(checkoutStatus string, approved bool) string {
	if approved {
		return "succeeded"
	}

	statusMap := map[string]string{
		"Pending":    "pending",
		"Authorized": "processing",
		"Captured":   "succeeded",
		"Declined":   "failed",
		"Canceled":   "canceled",
		"Voided":     "canceled",
	}

	if status, ok := statusMap[checkoutStatus]; ok {
		return status
	}

	return "pending"
}

// mapCheckoutCardBrand maps Checkout.com card scheme to standard brand
func mapCheckoutCardBrand(scheme string) string {
	brandMap := map[string]string{
		"VISA":             "visa",
		"MASTERCARD":       "mastercard",
		"AMERICAN_EXPRESS": "amex",
		"AMEX":             "amex",
	}

	if brand, ok := brandMap[strings.ToUpper(scheme)]; ok {
		return brand
	}

	return "other"
}

// mapCheckoutWebhookEvent maps Checkout.com event to standard event type
func mapCheckoutWebhookEvent(checkoutEvent string) string {
	eventMap := map[string]string{
		"payment_approved": "payment.succeeded",
		"payment_declined": "payment.failed",
		"payment_pending":  "payment.pending",
		"payment_captured": "payment.succeeded",
		"payment_voided":   "payment.canceled",
		"payment_refunded": "refund.succeeded",
	}

	if event, ok := eventMap[checkoutEvent]; ok {
		return event
	}

	return "payment.pending"
}

// mapCheckoutWebhookStatus maps Checkout.com webhook status
func mapCheckoutWebhookStatus(status string) string {
	return mapCheckoutStatus(status, false)
}
