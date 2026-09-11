package payment

import (
	"github.com/canakyuz/keystone/examples/verticals/usecase/payment"
	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles webhook HTTP requests from payment providers
type WebhookHandler struct {
	paymentService *payment.Service
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(paymentService *payment.Service) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
	}
}

// HandleWebhook handles incoming webhook from payment providers
// POST /api/v1/webhooks/payment/:provider
// This endpoint is public (no auth required)
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	provider := c.Params("provider")

	// Read raw body
	body := c.Body()

	// Get signature from header (provider-specific)
	var signature string
	switch provider {
	case "iyzico":
		signature = c.Get("X-IYZ-Signature", "")
	case "checkout":
		signature = c.Get("Cko-Signature", "")
	case "stripe":
		signature = c.Get("Stripe-Signature", "")
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unknown payment provider",
		})
	}

	// Get IP address
	ipAddress := c.IP()

	// Get request headers
	headers := make(map[string]any)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Extract tenant ID from webhook payload or query param
	// In production, this should be derived from the webhook payload
	// or from a dedicated webhook URL per tenant
	tenantID := c.Query("tenant_id", "")
	if tenantID == "" {
		// Try to extract from payload (provider-specific logic)
		tenantID = extractTenantIDFromPayload(provider, body)
	}

	if tenantID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tenant ID not found in webhook",
		})
	}

	// Process webhook
	result, err := h.paymentService.ProcessWebhook(
		c.Context(),
		tenantID,
		provider,
		body,
		signature,
		ipAddress,
		headers,
	)

	if err != nil {
		// Return 200 even on error to prevent provider retries
		// Log the error internally
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   result,
	})
}

// extractTenantIDFromPayload extracts tenant ID from webhook payload
// This is provider-specific logic
func extractTenantIDFromPayload(provider string, payload []byte) string {
	// In production, parse the payload and extract tenant ID
	// from metadata or custom fields

	// Example for iyzico:
	// Parse JSON and look for conversationId or metadata.tenant_id

	// Example for Checkout.com:
	// Parse JSON and look for metadata.tenant_id

	// For now, return empty string
	// This should be implemented based on actual webhook payload structure
	return ""
}

// HandleIyzicoWebhook handles iyzico-specific webhook
// POST /api/v1/webhooks/payment/iyzico/:tenant_id
func (h *WebhookHandler) HandleIyzicoWebhook(c *fiber.Ctx) error {
	tenantID := c.Params("tenant_id")

	body := c.Body()

	signature := c.Get("X-IYZ-Signature", "")
	ipAddress := c.IP()

	headers := make(map[string]any)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	result, err := h.paymentService.ProcessWebhook(
		c.Context(),
		tenantID,
		"iyzico",
		body,
		signature,
		ipAddress,
		headers,
	)

	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   result,
	})
}

// HandleCheckoutWebhook handles Checkout.com-specific webhook
// POST /api/v1/webhooks/payment/checkout/:tenant_id
func (h *WebhookHandler) HandleCheckoutWebhook(c *fiber.Ctx) error {
	tenantID := c.Params("tenant_id")

	body := c.Body()

	signature := c.Get("Cko-Signature", "")
	ipAddress := c.IP()

	headers := make(map[string]any)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	result, err := h.paymentService.ProcessWebhook(
		c.Context(),
		tenantID,
		"checkout",
		body,
		signature,
		ipAddress,
		headers,
	)

	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   result,
	})
}
