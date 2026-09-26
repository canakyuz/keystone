// Package upload serves the file upload routes. The rules for what is accepted and where it
// is stored live in internal/usecase/upload.
package upload

import (
	"context"
	"errors"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/internal/middleware"
	uploadrepo "github.com/canakyuz/keystone/internal/repository/upload"
	uploaduc "github.com/canakyuz/keystone/internal/usecase/upload"
	"github.com/canakyuz/keystone/pkg/logger"
)

// Storage is what the handler needs from the upload usecase.
type Storage interface {
	Accept(ctx context.Context, tenantID string, category uploaduc.Category, file *multipart.FileHeader) (*uploadrepo.Record, error)
}

// Handler handles file upload requests
type Handler struct {
	logger  *logger.Logger
	storage Storage
}

// NewHandler creates a new upload handler
func NewHandler(log *logger.Logger, storage Storage) *Handler {
	return &Handler{logger: log, storage: storage}
}

// UploadLogo handles logo file upload
// POST /api/v1/upload/logo
func (h *Handler) UploadLogo(c *fiber.Ctx) error {
	return h.accept(c, uploaduc.Logo)
}

// UploadFavicon handles favicon file upload
// POST /api/v1/upload/favicon
func (h *Handler) UploadFavicon(c *fiber.Ctx) error {
	return h.accept(c, uploaduc.Favicon)
}

// UploadImage handles general image upload
// POST /api/v1/upload/image
func (h *Handler) UploadImage(c *fiber.Ctx) error {
	return h.accept(c, uploaduc.Image)
}

func (h *Handler) accept(c *fiber.Ctx, category uploaduc.Category) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: tenant ID required",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file provided",
		})
	}

	record, err := h.storage.Accept(c.UserContext(), tenantID, category, file)

	var rejected *uploaduc.RejectedError
	switch {
	case errors.As(err, &rejected):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": rejected.Reason})
	case err != nil:
		h.logger.ErrorWithErr(err, "failed to store "+category.Name+" file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":           record.ID,
			"url":          record.Path,
			"filename":     file.Filename,
			"size":         file.Size,
			"content_type": record.ContentType,
		},
	})
}
