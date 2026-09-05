package tenant

import (
	"strconv"

	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/canakyuz/keystone/internal/usecase/tenant"
	"github.com/gofiber/fiber/v2"
)

// Handler handles tenant HTTP requests
type Handler struct {
	tenantService *tenant.Service
}

// NewHandler creates a new tenant handler
func NewHandler(tenantService *tenant.Service) *Handler {
	return &Handler{
		tenantService: tenantService,
	}
}

// Create creates a new tenant
// POST /api/v1/tenants
func (h *Handler) Create(c *fiber.Ctx) error {
	var req tenant.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.tenantService.Create(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}

// GetByID retrieves a tenant by ID
// GET /api/v1/tenants/:id
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := h.tenantService.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// GetBySlug retrieves a tenant by slug
// GET /api/v1/tenants/slug/:slug
func (h *Handler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	result, err := h.tenantService.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// GetCurrent retrieves current tenant (from auth context)
// GET /api/v1/tenants/current
func (h *Handler) GetCurrent(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	result, err := h.tenantService.GetByID(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// List retrieves all tenants with pagination
// GET /api/v1/tenants
func (h *Handler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	status := c.Query("status")
	plan := c.Query("plan")
	search := c.Query("search")

	result, err := h.tenantService.List(c.Context(), page, perPage, status, plan, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// Update updates a tenant
// PATCH /api/v1/tenants/:id
func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req tenant.UpdateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.tenantService.Update(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// Suspend suspends a tenant
// POST /api/v1/tenants/:id/suspend
func (h *Handler) Suspend(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.tenantService.Suspend(c.Context(), id, req.Reason); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tenant suspended successfully",
	})
}

// Activate activates a tenant
// POST /api/v1/tenants/:id/activate
func (h *Handler) Activate(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.tenantService.Activate(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tenant activated successfully",
	})
}

// UpgradePlan upgrades tenant subscription plan
// POST /api/v1/tenants/:id/upgrade
func (h *Handler) UpgradePlan(c *fiber.Ctx) error {
	id := c.Params("id")

	var req tenant.UpgradePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.tenantService.UpgradePlan(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// SetCustomDomain sets custom domain for tenant
// POST /api/v1/tenants/:id/domain
func (h *Handler) SetCustomDomain(c *fiber.Ctx) error {
	id := c.Params("id")

	var req tenant.SetCustomDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.tenantService.SetCustomDomain(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// VerifyCustomDomain verifies custom domain
// POST /api/v1/tenants/:id/domain/verify
func (h *Handler) VerifyCustomDomain(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := h.tenantService.VerifyCustomDomain(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// Delete deletes a tenant
// DELETE /api/v1/tenants/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.tenantService.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// UpdateBranding updates tenant branding settings
// PATCH /api/v1/tenants/:id/branding
func (h *Handler) UpdateBranding(c *fiber.Ctx) error {
	id := c.Params("id")

	var req tenant.UpdateBrandingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.tenantService.UpdateBranding(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// GetStats retrieves tenant statistics
// GET /api/v1/tenants/stats
func (h *Handler) GetStats(c *fiber.Ctx) error {
	stats, err := h.tenantService.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": stats,
	})
}
