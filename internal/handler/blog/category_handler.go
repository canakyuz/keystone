package blog

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	blogUsecase "nexpaces-api/internal/usecase/blog"
	"nexpaces-api/pkg/logger"
)

type CategoryHandler struct {
	service *blogUsecase.CategoryService
	logger  logger.Logger
}

func NewCategoryHandler(service *blogUsecase.CategoryService, logger logger.Logger) *CategoryHandler {
	return &CategoryHandler{
		service: service,
		logger:  logger,
	}
}

func (h *CategoryHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req blogUsecase.CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	category, err := h.service.Create(c.Context(), tenantID, req)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to create category")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create category"})
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}

func (h *CategoryHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	category, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Category not found"})
	}

	return c.JSON(category)
}

func (h *CategoryHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	category, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Category not found"})
	}

	return c.JSON(category)
}

func (h *CategoryHandler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	categories, err := h.service.List(c.Context(), tenantID, page, limit)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to list categories")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list categories"})
	}

	return c.JSON(categories)
}

func (h *CategoryHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req blogUsecase.UpdateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	category, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to update category")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update category"})
	}

	return c.JSON(category)
}

func (h *CategoryHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to delete category")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete category"})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
