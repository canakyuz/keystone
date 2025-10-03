package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nexpaces-api/internal/config"
)

// IyzicoProvider implements iyzico payment integration for Turkey
type IyzicoProvider struct {
	config *config.IyzicoConfig
	client *http.Client
}

// NewIyzicoProvider creates a new iyzico provider
func NewIyzicoProvider(cfg *config.IyzicoConfig) *IyzicoProvider {
	return &IyzicoProvider{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName returns the provider name
func (p *IyzicoProvider) GetName() string {
	return "iyzico"
}

// CreatePayment initiates a payment with iyzico
func (p *IyzicoProvider) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	// Build iyzico payment request
	iyzicoReq := map[string]interface{}{
		"locale":         "tr",
		"conversationId": req.ReferenceID,
		"price":          fmt.Sprintf("%.2f", req.Amount),
		"paidPrice":      fmt.Sprintf("%.2f", req.Amount),
		"currency":       req.Currency,
		"installment":    req.Installment,
		"basketId":       req.OrderID,
		"paymentChannel": "WEB",
		"paymentGroup":   "PRODUCT",

		// Callback URLs for 3DS
		"callbackUrl": req.CallbackURL,

		// Payment card
		"paymentCard": map[string]interface{}{
			"cardHolderName": req.CardHolderName,
			"cardNumber":     req.CardNumber,
			"expireMonth":    req.CardExpMonth,
			"expireYear":     req.CardExpYear,
			"cvc":            req.CardCVV,
		},

		// Buyer information
		"buyer": map[string]interface{}{
			"id":                  req.TenantID,
			"name":                req.CustomerFirstName,
			"surname":             req.CustomerLastName,
			"email":               req.CustomerEmail,
			"identityNumber":      "11111111111", // Required by iyzico
			"registrationAddress": req.BillingAddressLine,
			"ip":                  req.CustomerIPAddress,
			"city":                req.BillingCity,
			"country":             req.BillingCountry,
			"zipCode":             req.BillingZipCode,
		},

		// Billing and shipping address (same for digital goods)
		"billingAddress": map[string]interface{}{
			"contactName": req.CustomerFirstName + " " + req.CustomerLastName,
			"city":        req.BillingCity,
			"country":     req.BillingCountry,
			"address":     req.BillingAddressLine,
			"zipCode":     req.BillingZipCode,
		},

		"shippingAddress": map[string]interface{}{
			"contactName": req.CustomerFirstName + " " + req.CustomerLastName,
			"city":        req.BillingCity,
			"country":     req.BillingCountry,
			"address":     req.BillingAddressLine,
			"zipCode":     req.BillingZipCode,
		},

		// Basket items (required)
		"basketItems": []map[string]interface{}{
			{
				"id":        req.OrderID,
				"name":      req.Description,
				"category1": "Digital Services",
				"itemType":  "VIRTUAL",
				"price":     fmt.Sprintf("%.2f", req.Amount),
			},
		},
	}

	// Make API request
	respData, err := p.makeRequest(ctx, "/payment/3dsecure/initialize", iyzicoReq)
	if err != nil {
		return nil, fmt.Errorf("iyzico API error: %w", err)
	}

	// Parse response
	status, _ := respData["status"].(string)
	if status == "failure" {
		errorMessage, _ := respData["errorMessage"].(string)
		return &PaymentResponse{
			Status:         "failed",
			FailureCode:    "iyzico_error",
			FailureMessage: errorMessage,
			RawResponse:    respData,
		}, nil
	}

	// Extract 3DS HTML content
	threeDSHtmlContent, _ := respData["threeDSHtmlContent"].(string)
	paymentId, _ := respData["paymentId"].(string)

	// Determine if 3DS is required
	requires3DS := threeDSHtmlContent != ""

	response := &PaymentResponse{
		ProviderPaymentID: paymentId,
		Status:            "requires_3ds",
		Amount:            req.Amount,
		Currency:          req.Currency,
		Requires3DS:       requires3DS,
		ThreeDSVersion:    "2.0",
		ThreeDSHTMLContent: threeDSHtmlContent,
		Installment:       req.Installment,
		RawResponse:       respData,
	}

	// If no 3DS required, payment is completed
	if !requires3DS {
		response.Status = "succeeded"
	}

	return response, nil
}

// CompletePayment completes 3DS authentication
func (p *IyzicoProvider) CompletePayment(ctx context.Context, req *CompletePaymentRequest) (*PaymentResponse, error) {
	// Build completion request
	iyzicoReq := map[string]interface{}{
		"locale":         "tr",
		"conversationId": req.PaymentID,
		"paymentId":      req.ProviderPaymentID,
		"conversationData": req.ConversationData,
	}

	// Make API request
	respData, err := p.makeRequest(ctx, "/payment/3dsecure/auth", iyzicoReq)
	if err != nil {
		return nil, fmt.Errorf("iyzico 3DS completion error: %w", err)
	}

	// Parse response
	status, _ := respData["status"].(string)
	if status == "failure" {
		errorMessage, _ := respData["errorMessage"].(string)
		return &PaymentResponse{
			ProviderPaymentID: req.ProviderPaymentID,
			Status:            "failed",
			FailureCode:       "3ds_failed",
			FailureMessage:    errorMessage,
			RawResponse:       respData,
		}, nil
	}

	// Extract payment details
	paymentId, _ := respData["paymentId"].(string)
	paidPrice, _ := respData["paidPrice"].(float64)
	currency, _ := respData["currency"].(string)
	installment, _ := respData["installment"].(float64)

	// Extract card information
	var cardBrand, cardLast4, cardBin string
	if cardDetails, ok := respData["cardDetails"].(map[string]interface{}); ok {
		cardBrand, _ = cardDetails["cardFamily"].(string)
		cardLast4, _ = cardDetails["lastFourDigits"].(string)
		cardBin, _ = cardDetails["binNumber"].(string)
	}

	response := &PaymentResponse{
		ProviderPaymentID: paymentId,
		Status:            "succeeded",
		Amount:            paidPrice,
		Currency:          currency,
		Installment:       int(installment),
		CardBrand:         mapIyzicoCardBrand(cardBrand),
		CardLast4:         cardLast4,
		CardBIN:           cardBin,
		RawResponse:       respData,
	}

	return response, nil
}

// GetPayment retrieves payment details
func (p *IyzicoProvider) GetPayment(ctx context.Context, providerPaymentID string) (*PaymentResponse, error) {
	// Build request
	iyzicoReq := map[string]interface{}{
		"locale":         "tr",
		"conversationId": providerPaymentID,
		"paymentId":      providerPaymentID,
	}

	// Make API request
	respData, err := p.makeRequest(ctx, "/payment/detail", iyzicoReq)
	if err != nil {
		return nil, fmt.Errorf("iyzico get payment error: %w", err)
	}

	// Parse response
	status, _ := respData["status"].(string)
	if status == "failure" {
		return nil, fmt.Errorf("payment not found")
	}

	paymentStatus, _ := respData["paymentStatus"].(string)
	paidPrice, _ := respData["paidPrice"].(float64)
	currency, _ := respData["currency"].(string)

	return &PaymentResponse{
		ProviderPaymentID: providerPaymentID,
		Status:            mapIyzicoStatus(paymentStatus),
		Amount:            paidPrice,
		Currency:          currency,
		RawResponse:       respData,
	}, nil
}

// CancelPayment cancels a payment
func (p *IyzicoProvider) CancelPayment(ctx context.Context, providerPaymentID string) error {
	// Build request
	iyzicoReq := map[string]interface{}{
		"locale":         "tr",
		"conversationId": providerPaymentID,
		"paymentId":      providerPaymentID,
		"ip":             "127.0.0.1",
	}

	// Make API request
	respData, err := p.makeRequest(ctx, "/payment/cancel", iyzicoReq)
	if err != nil {
		return fmt.Errorf("iyzico cancel error: %w", err)
	}

	// Check status
	status, _ := respData["status"].(string)
	if status == "failure" {
		errorMessage, _ := respData["errorMessage"].(string)
		return fmt.Errorf("cancel failed: %s", errorMessage)
	}

	return nil
}

// CreateRefund initiates a refund
func (p *IyzicoProvider) CreateRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// Build request
	iyzicoReq := map[string]interface{}{
		"locale":         "tr",
		"conversationId": req.PaymentID,
		"paymentTransactionId": req.ProviderPaymentID,
		"price":          fmt.Sprintf("%.2f", req.Amount),
		"ip":             "127.0.0.1",
		"currency":       req.Currency,
	}

	// Make API request
	respData, err := p.makeRequest(ctx, "/payment/refund", iyzicoReq)
	if err != nil {
		return nil, fmt.Errorf("iyzico refund error: %w", err)
	}

	// Parse response
	status, _ := respData["status"].(string)
	if status == "failure" {
		errorMessage, _ := respData["errorMessage"].(string)
		return &RefundResponse{
			Status:         "failed",
			FailureCode:    "refund_failed",
			FailureMessage: errorMessage,
			Amount:         req.Amount,
			Currency:       req.Currency,
			RawResponse:    respData,
		}, nil
	}

	paymentId, _ := respData["paymentId"].(string)
	price, _ := respData["price"].(float64)

	return &RefundResponse{
		ProviderRefundID: paymentId,
		Status:           "succeeded",
		Amount:           price,
		Currency:         req.Currency,
		RawResponse:      respData,
	}, nil
}

// GetRefund retrieves refund details
func (p *IyzicoProvider) GetRefund(ctx context.Context, providerRefundID string) (*RefundResponse, error) {
	// iyzico doesn't have a separate refund detail endpoint
	// Refund status is included in payment details
	return nil, fmt.Errorf("iyzico does not support separate refund retrieval")
}

// VerifyWebhookSignature verifies webhook signature
func (p *IyzicoProvider) VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error {
	// iyzico webhook signature verification
	expectedSignature := p.generateSignature(string(payload))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid webhook signature")
	}

	return nil
}

// ParseWebhook parses webhook payload
func (p *IyzicoProvider) ParseWebhook(ctx context.Context, payload []byte) (*WebhookEvent, error) {
	var webhookData map[string]interface{}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Extract event details
	eventType, _ := webhookData["status"].(string)
	paymentId, _ := webhookData["paymentId"].(string)
	paidPrice, _ := webhookData["paidPrice"].(float64)
	currency, _ := webhookData["currency"].(string)

	event := &WebhookEvent{
		ProviderEventID: paymentId,
		EventType:       mapIyzicoWebhookEvent(eventType),
		PaymentID:       paymentId,
		Status:          mapIyzicoStatus(eventType),
		Amount:          paidPrice,
		Currency:        currency,
		Payload:         webhookData,
	}

	return event, nil
}

// makeRequest makes HTTP request to iyzico API
func (p *IyzicoProvider) makeRequest(ctx context.Context, endpoint string, body interface{}) (map[string]interface{}, error) {
	// Marshal body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := p.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Generate authorization header
	authHeader := p.generateAuthorizationHeader()
	req.Header.Set("Authorization", authHeader)

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

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// generateAuthorizationHeader generates iyzico authorization header
func (p *IyzicoProvider) generateAuthorizationHeader() string {
	randomString := generateRandomString()
	dataToEncrypt := p.config.APIKey + ":" + randomString

	hash := hmac.New(sha256.New, []byte(p.config.SecretKey))
	hash.Write([]byte(dataToEncrypt))
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	return fmt.Sprintf("IYZWS %s:%s", p.config.APIKey, signature)
}

// generateSignature generates HMAC signature
func (p *IyzicoProvider) generateSignature(data string) string {
	hash := hmac.New(sha256.New, []byte(p.config.SecretKey))
	hash.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// generateRandomString generates a random string for auth header
func generateRandomString() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// mapIyzicoStatus maps iyzico status to standard status
func mapIyzicoStatus(iyzicoStatus string) string {
	statusMap := map[string]string{
		"SUCCESS":    "succeeded",
		"FAILURE":    "failed",
		"INIT_THREEDS": "requires_3ds",
		"PENDING":    "pending",
	}

	if status, ok := statusMap[strings.ToUpper(iyzicoStatus)]; ok {
		return status
	}

	return "pending"
}

// mapIyzicoCardBrand maps iyzico card brand to standard brand
func mapIyzicoCardBrand(iyzicoCardBrand string) string {
	brandMap := map[string]string{
		"VISA":             "visa",
		"MASTER_CARD":      "mastercard",
		"AMERICAN_EXPRESS": "amex",
		"TROY":             "troy",
	}

	if brand, ok := brandMap[strings.ToUpper(iyzicoCardBrand)]; ok {
		return brand
	}

	return "other"
}

// mapIyzicoWebhookEvent maps iyzico event to standard event type
func mapIyzicoWebhookEvent(iyzicoEvent string) string {
	eventMap := map[string]string{
		"SUCCESS": "payment.succeeded",
		"FAILURE": "payment.failed",
		"PENDING": "payment.pending",
	}

	if event, ok := eventMap[strings.ToUpper(iyzicoEvent)]; ok {
		return event
	}

	return "payment.pending"
}
