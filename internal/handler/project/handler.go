package project

import (
	"github.com/gofiber/fiber/v2"

	projectUsecase "nexspaces-api/internal/usecase/project"
)

// Handler handles HTTP requests for projects
type Handler struct {
	service *projectUsecase.Service
}

// NewHandler creates a new project handler
func NewHandler(service *projectUsecase.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Create handles project creation
// POST /api/v1/projects
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req projectUsecase.CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// GetByID handles retrieving a project by ID
// GET /api/v1/projects/:id
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	project, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// GetBySlug handles retrieving a project by slug
// GET /api/v1/projects/slug/:slug
func (h *Handler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	project, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// List handles listing projects
// GET /api/v1/projects
func (h *Handler) List(c *fiber.Ctx) error {
	var req projectUsecase.ListProjectsRequest

	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	response, err := h.service.List(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// Update handles updating a project
// PATCH /api/v1/projects/:id
func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req projectUsecase.UpdateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// Delete handles deleting a project
// DELETE /api/v1/projects/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// MarkAsCompleted handles marking project as completed
// POST /api/v1/projects/:id/complete
func (h *Handler) MarkAsCompleted(c *fiber.Ctx) error {
	id := c.Params("id")

	project, err := h.service.MarkAsCompleted(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// SetFeatured handles setting/unsetting project as featured
// POST /api/v1/projects/:id/featured
func (h *Handler) SetFeatured(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Featured bool `json:"featured"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.SetFeatured(c.Context(), id, req.Featured)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// GetFeatured handles retrieving featured projects
// GET /api/v1/projects/featured
func (h *Handler) GetFeatured(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)

	projects, err := h.service.GetFeatured(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    projects,
	})
}

// GetByCategory handles retrieving projects by category
// GET /api/v1/projects/category/:category
func (h *Handler) GetByCategory(c *fiber.Ctx) error {
	category := c.Params("category")
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 20)

	response, err := h.service.GetByCategory(c.Context(), category, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// GetStats handles retrieving project statistics
// GET /api/v1/projects/stats
func (h *Handler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// AddImage handles adding an image to project
// POST /api/v1/projects/:id/images
func (h *Handler) AddImage(c *fiber.Ctx) error {
	id := c.Params("id")

	var req projectUsecase.AddImageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.AddImage(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// RemoveImage handles removing an image from project
// DELETE /api/v1/projects/:id/images
func (h *Handler) RemoveImage(c *fiber.Ctx) error {
	id := c.Params("id")

	var req projectUsecase.RemoveImageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.RemoveImage(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// AddTechnology handles adding a technology to project
// POST /api/v1/projects/:id/technologies
func (h *Handler) AddTechnology(c *fiber.Ctx) error {
	id := c.Params("id")

	var req projectUsecase.AddTechnologyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.AddTechnology(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}

// RemoveTechnology handles removing a technology from project
// DELETE /api/v1/projects/:id/technologies
func (h *Handler) RemoveTechnology(c *fiber.Ctx) error {
	id := c.Params("id")

	var req projectUsecase.RemoveTechnologyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.service.RemoveTechnology(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    project,
	})
}
