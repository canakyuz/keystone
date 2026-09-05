package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// File size limits
	MaxLogoSize    = 2 * 1024 * 1024  // 2MB
	MaxFaviconSize = 500 * 1024       // 500KB
	MaxFileSize    = 10 * 1024 * 1024 // 10MB (general)

	// Upload directories
	UploadBasePath = "./uploads"
)

// AllowedFileTypes defines allowed MIME types for different file categories
var AllowedFileTypes = map[string][]string{
	"logo": {
		"image/png",
		"image/jpeg",
		"image/svg+xml",
		"image/webp",
	},
	"favicon": {
		"image/x-icon",
		"image/png",
		"image/vnd.microsoft.icon",
	},
	"image": {
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
		"image/webp",
		"image/svg+xml",
	},
}

// Handler handles file upload requests
type Handler struct {
	logger     *logger.Logger
	uploadPath string
}

// NewHandler creates a new upload handler
func NewHandler(log *logger.Logger) *Handler {
	return &Handler{
		logger:     log,
		uploadPath: UploadBasePath,
	}
}

// UploadLogo handles logo file upload
// POST /api/v1/upload/logo
func (h *Handler) UploadLogo(c *fiber.Ctx) error {
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

	// Validate file
	if err := h.validateFile(file, "logo", MaxLogoSize); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Save file
	filePath, err := h.saveFile(c, file, tenantID, "logo")
	if err != nil {
		h.logger.ErrorWithErr(err, "failed to save logo file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"url":      filePath,
			"filename": file.Filename,
			"size":     file.Size,
		},
	})
}

// UploadFavicon handles favicon file upload
// POST /api/v1/upload/favicon
func (h *Handler) UploadFavicon(c *fiber.Ctx) error {
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

	// Validate file
	if err := h.validateFile(file, "favicon", MaxFaviconSize); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Save file
	filePath, err := h.saveFile(c, file, tenantID, "favicon")
	if err != nil {
		h.logger.ErrorWithErr(err, "failed to save favicon file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"url":      filePath,
			"filename": file.Filename,
			"size":     file.Size,
		},
	})
}

// UploadImage handles general image upload
// POST /api/v1/upload/image
func (h *Handler) UploadImage(c *fiber.Ctx) error {
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

	// Validate file
	if err := h.validateFile(file, "image", MaxFileSize); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Save file
	filePath, err := h.saveFile(c, file, tenantID, "images")
	if err != nil {
		h.logger.ErrorWithErr(err, "failed to save image file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"url":      filePath,
			"filename": file.Filename,
			"size":     file.Size,
		},
	})
}

// validateFile validates file type and size
func (h *Handler) validateFile(file *multipart.FileHeader, category string, maxSize int64) error {
	// Check file size
	if file.Size > maxSize {
		return fmt.Errorf("file size exceeds limit (%d bytes)", maxSize)
	}

	// Check file type
	allowedTypes, exists := AllowedFileTypes[category]
	if !exists {
		return fmt.Errorf("invalid file category: %s", category)
	}

	// Open file to check MIME type
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type
	contentType := file.Header.Get("Content-Type")

	// Validate against allowed types
	allowed := false
	for _, allowedType := range allowedTypes {
		if strings.HasPrefix(contentType, allowedType) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type not allowed: %s (allowed: %v)", contentType, allowedTypes)
	}

	return nil
}

// saveFile saves uploaded file to disk (tenant-scoped)
func (h *Handler) saveFile(c *fiber.Ctx, file *multipart.FileHeader, tenantID, subdir string) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Create tenant-specific directory path
	// Format: uploads/tenants/{tenant_id}/{subdir}/
	dirPath := filepath.Join(h.uploadPath, "tenants", tenantID, subdir)

	// Create directory if not exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Full file path
	fullPath := filepath.Join(dirPath, filename)

	// Save file
	if err := c.SaveFile(file, fullPath); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative URL path
	relativePath := filepath.Join("/uploads/tenants", tenantID, subdir, filename)

	h.logger.WithFields(logger.Fields{
		"tenant_id": tenantID,
		"file_path": relativePath,
		"file_size": file.Size,
	}).Info("File uploaded successfully")

	return relativePath, nil
}
