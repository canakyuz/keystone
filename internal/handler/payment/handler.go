package payment

import (
	"strconv"

	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/canakyuz/keystone/internal/usecase/payment"
	"github.com/gofiber/fiber/v2"
)

// Handler handles payment HTTP requests
type Handler struct {
	paymentService *payment.Service
}

// NewHandler creates a new payment handler
func NewHandler(paymentService *payment.Service) *Handler {
	return &Handler{
		paymentService: paymentService,
	}
}

// CreatePayment creates a new payment
// POST /api/v1/payments
func (h *Handler) CreatePayment(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	// Get tenant settings (would be from tenant service in production)
	tenantSettings := make(map[string]interface{})
	if settings := c.Locals("tenant_settings"); settings != nil {
		if settingsMap, ok := settings.(map[string]interface{}); ok {
			tenantSettings = settingsMap
		}
	}

	var req payment.CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.paymentService.CreatePayment(c.Context(), tenantID, userID, tenantSettings, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}

// Complete3DSPayment completes 3DS authentication
// POST /api/v1/payments/complete-3ds
func (h *Handler) Complete3DSPayment(c *fiber.Ctx) error {
	var req payment.Complete3DSRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.paymentService.Complete3DSPayment(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// GetPayment retrieves a payment by ID
// GET /api/v1/payments/:id
func (h *Handler) GetPayment(c *fiber.Ctx) error {
	paymentID := c.Params("id")

	result, err := h.paymentService.GetPayment(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// ListPayments lists payments for current tenant
// GET /api/v1/payments
func (h *Handler) ListPayments(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	req := &payment.ListPaymentsRequest{
		Limit:  limit,
		Offset: offset,
	}

	results, err := h.paymentService.ListPayments(c.Context(), tenantID, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": results,
	})
}

// CreateRefund creates a refund for a payment
// POST /api/v1/payments/:id/refund
func (h *Handler) CreateRefund(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	paymentID := c.Params("id")

	var req payment.CreateRefundRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	req.PaymentID = paymentID

	result, err := h.paymentService.CreateRefund(c.Context(), tenantID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}
