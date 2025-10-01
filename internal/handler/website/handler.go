package website

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"nexpaces-api/internal/domain/website"
	websiteUsecase "nexpaces-api/internal/usecase/website"
)

// Handler handles HTTP requests for websites
type Handler struct {
	service *websiteUsecase.Service
}

// NewHandler creates a new website handler
func NewHandler(service *websiteUsecase.Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /websites
func (h *Handler) List(c *fiber.Ctx) error {
	// Get tenant ID from context (set by middleware)
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	req := websiteUsecase.ListWebsitesRequest{
		TenantID: tenantID,
	}

	websites, err := h.service.List(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(fiber.Map{
		"items": websites,
		"total": len(websites),
	})
}

// Create handles POST /websites
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	var req websiteUsecase.CreateWebsiteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Override tenant ID from context (security measure)
	req.TenantID = tenantID

	site, err := h.service.Create(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(site)
}

// GetByID handles GET /websites/:id
func (h *Handler) GetByID(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid website ID",
		})
	}

	req := websiteUsecase.GetWebsiteRequest{
		ID:       id,
		TenantID: tenantID,
	}

	site, err := h.service.GetByID(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(site)
}

// GetBySlug handles GET /websites/slug/:slug (public endpoint)
func (h *Handler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "slug is required",
		})
	}

	site, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(site)
}

// Update handles PATCH /websites/:id
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid website ID",
		})
	}

	var req websiteUsecase.UpdateWebsiteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	site, err := h.service.Update(c.Context(), id, tenantID, req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(site)
}

// Publish handles POST /websites/:id/publish
func (h *Handler) Publish(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid website ID",
		})
	}

	req := websiteUsecase.PublishWebsiteRequest{
		ID:       id,
		TenantID: tenantID,
	}

	site, err := h.service.Publish(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(site)
}

// Archive handles POST /websites/:id/archive
func (h *Handler) Archive(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid website ID",
		})
	}

	req := websiteUsecase.ArchiveWebsiteRequest{
		ID:       id,
		TenantID: tenantID,
	}

	site, err := h.service.Archive(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(site)
}

// Delete handles DELETE /websites/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing tenant context",
		})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid website ID",
		})
	}

	req := websiteUsecase.DeleteWebsiteRequest{
		ID:       id,
		TenantID: tenantID,
	}

	if err := h.service.Delete(c.Context(), req); err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// getTenantID extracts tenant ID from Fiber context (set by middleware)
func getTenantID(c *fiber.Ctx) (uuid.UUID, error) {
	tenantIDStr := c.Locals("tenant_id")
	if tenantIDStr == nil {
		return uuid.Nil, errors.New("missing tenant_id")
	}

	switch v := tenantIDStr.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, errors.New("invalid tenant_id type")
	}
}

// handleError converts domain errors to HTTP responses
func handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, website.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "website not found",
		})
	case errors.Is(err, website.ErrSlugExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "website slug already exists",
		})
	case errors.Is(err, website.ErrCannotPublish):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot publish website: missing homepage or invalid status",
		})
	case errors.Is(err, website.ErrAlreadyArchived):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "website is already archived",
		})
	case errors.Is(err, website.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid input",
		})
	case errors.Is(err, website.ErrUnauthorized):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "unauthorized access",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
}
