package blog

import (
	"strconv"

	blogUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/blog"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

type PostHandler struct {
	service *blogUsecase.PostService
	logger  logger.Logger
}

func NewPostHandler(service *blogUsecase.PostService, logger logger.Logger) *PostHandler {
	return &PostHandler{
		service: service,
		logger:  logger,
	}
}

func (h *PostHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var req blogUsecase.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	post, err := h.service.Create(c.Context(), tenantID, userID, req)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to create post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create post"})
	}

	return c.Status(fiber.StatusCreated).JSON(post)
}

func (h *PostHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	post, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Post not found"})
	}

	return c.JSON(post)
}

func (h *PostHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	post, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Post not found"})
	}

	// Increment view count asynchronously
	go func() {
		_ = h.service.IncrementViewCount(c.Context(), post.ID)
	}()

	return c.JSON(post)
}

func (h *PostHandler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	var categoryID, status *string
	var featured *bool

	if cat := c.Query("category_id"); cat != "" {
		categoryID = &cat
	}
	if st := c.Query("status"); st != "" {
		status = &st
	}
	if feat := c.Query("featured"); feat != "" {
		f := feat == "true"
		featured = &f
	}

	posts, err := h.service.List(c.Context(), tenantID, categoryID, status, featured, page, limit)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to list posts")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list posts"})
	}

	return c.JSON(posts)
}

func (h *PostHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(string)

	var req blogUsecase.UpdatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	post, err := h.service.Update(c.Context(), id, userID, req)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to update post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update post"})
	}

	return c.JSON(post)
}

func (h *PostHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to delete post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete post"})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *PostHandler) Publish(c *fiber.Ctx) error {
	id := c.Params("id")

	post, err := h.service.Publish(c.Context(), id)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to publish post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to publish post"})
	}

	return c.JSON(post)
}

func (h *PostHandler) Archive(c *fiber.Ctx) error {
	id := c.Params("id")

	post, err := h.service.Archive(c.Context(), id)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to archive post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to archive post"})
	}

	return c.JSON(post)
}

func (h *PostHandler) GetByTag(c *fiber.Ctx) error {
	tag := c.Params("tag")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	posts, err := h.service.GetByTag(c.Context(), tag, page, limit)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to get posts by tag")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get posts by tag"})
	}

	return c.JSON(posts)
}

func (h *PostHandler) GetFeatured(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	posts, err := h.service.GetFeatured(c.Context(), page, limit)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to get featured posts")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get featured posts"})
	}

	return c.JSON(posts)
}

func (h *PostHandler) GetByCategoryID(c *fiber.Ctx) error {
	categoryID := c.Params("category_id")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	posts, err := h.service.GetByCategoryID(c.Context(), categoryID, page, limit)
	if err != nil {
		h.logger.WithFields(logger.Fields{"error": err.Error()}).Error("Failed to get posts by category")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get posts by category"})
	}

	return c.JSON(posts)
}
