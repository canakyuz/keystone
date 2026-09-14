package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/canakyuz/keystone/internal/middleware"
	uploadrepo "github.com/canakyuz/keystone/internal/repository/upload"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Upload size limits, in bytes, and where accepted files are stored.
//
// Each limit has to fit inside the request body limit (MAX_REQUEST_SIZE, 4MB by default)
// with room for the multipart framing around the file. The general limit used to be 10MB,
// which no request could reach: the body was refused before the handler ran.
const (
	// File size limits
	MaxLogoSize    = 2 * 1024 * 1024 // 2MB
	MaxFaviconSize = 500 * 1024      // 500KB
	MaxFileSize    = 3 * 1024 * 1024 // 3MB (general)

	// Upload directories
	UploadBasePath = "./uploads"
)

// AllowedFileTypes lists, per category, the content types a file may be stored as. The type
// is the one read from the file's own first bytes, never the one the client declares.
//
// SVG is not among them. An SVG is a document that can carry script, served from this
// origin it runs with this origin's authority, and no byte signature tells a harmless one
// from a hostile one.
var AllowedFileTypes = map[string][]string{
	"logo":    {"image/png", "image/jpeg", "image/webp"},
	"favicon": {"image/x-icon", "image/png"},
	"image":   {"image/png", "image/jpeg", "image/gif", "image/webp"},
}

// extensions gives the extension a file of each accepted type is stored under.
//
// The server chooses it. It used to come from the client's file name, so a page named
// x.html declared as image/png was stored as x.html and served back as text/html.
var extensions = map[string]string{
	"image/png":    ".png",
	"image/jpeg":   ".jpg",
	"image/gif":    ".gif",
	"image/webp":   ".webp",
	"image/x-icon": ".ico",
}

// errTypeNotAllowed is the refusal a client sees for a file that is not an accepted image.
var errTypeNotAllowed = errors.New("file type not allowed")

// Recorder records a stored file. It is the one thing the handler needs from the upload
// repository.
type Recorder interface {
	Create(ctx context.Context, record *uploadrepo.Record) error
}

// Handler handles file upload requests
type Handler struct {
	logger     *logger.Logger
	records    Recorder
	uploadPath string
}

// NewHandler creates a new upload handler
func NewHandler(log *logger.Logger, records Recorder) *Handler {
	return &Handler{
		logger:     log,
		records:    records,
		uploadPath: UploadBasePath,
	}
}

// UploadLogo handles logo file upload
// POST /api/v1/upload/logo
func (h *Handler) UploadLogo(c *fiber.Ctx) error {
	return h.accept(c, "logo", "logo", MaxLogoSize)
}

// UploadFavicon handles favicon file upload
// POST /api/v1/upload/favicon
func (h *Handler) UploadFavicon(c *fiber.Ctx) error {
	return h.accept(c, "favicon", "favicon", MaxFaviconSize)
}

// UploadImage handles general image upload
// POST /api/v1/upload/image
func (h *Handler) UploadImage(c *fiber.Ctx) error {
	return h.accept(c, "image", "images", MaxFileSize)
}

// accept validates the request's file for category, stores it under subdir and records it.
func (h *Handler) accept(c *fiber.Ctx, category, subdir string, maxSize int64) error {
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

	contentType, err := h.validateFile(file, category, maxSize)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	urlPath, diskPath, err := h.saveFile(c, file, tenantID, subdir, contentType)
	if err != nil {
		h.logger.ErrorWithErr(err, "failed to save "+category+" file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	record := &uploadrepo.Record{
		TenantID:    tenantID,
		Category:    category,
		Path:        urlPath,
		ContentType: contentType,
		SizeBytes:   file.Size,
	}
	if err := h.records.Create(c.UserContext(), record); err != nil {
		// The file is on disk with no record of it. Removing it keeps the two in step: a
		// stored file nobody can account for is what the record exists to rule out.
		if removeErr := os.Remove(diskPath); removeErr != nil {
			h.logger.ErrorWithErr(removeErr, "failed to remove an unrecorded upload")
		}
		return err
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":           record.ID,
			"url":          urlPath,
			"filename":     file.Filename,
			"size":         file.Size,
			"content_type": contentType,
		},
	})
}

// validateFile checks the file's size and reads its type from its first bytes, returning
// that type when the category accepts it.
func (h *Handler) validateFile(file *multipart.FileHeader, category string, maxSize int64) (string, error) {
	if file.Size > maxSize {
		return "", fmt.Errorf("file size exceeds limit (%d bytes)", maxSize)
	}

	allowedTypes, exists := AllowedFileTypes[category]
	if !exists {
		return "", fmt.Errorf("invalid file category: %s", category)
	}

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// DetectContentType considers at most the first 512 bytes. A shorter file is read whole.
	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	contentType := http.DetectContentType(head[:n])
	if !slices.Contains(allowedTypes, contentType) {
		return "", fmt.Errorf("%w (allowed: %v)", errTypeNotAllowed, allowedTypes)
	}

	return contentType, nil
}

// saveFile saves uploaded file to disk (tenant-scoped), under a name and extension the
// server chose. It returns the path the file is served from and the path it was written to.
func (h *Handler) saveFile(c *fiber.Ctx, file *multipart.FileHeader, tenantID, subdir, contentType string) (string, string, error) {
	filename := uuid.New().String() + extensions[contentType]

	// Create tenant-specific directory path
	// Format: uploads/tenants/{tenant_id}/{subdir}/
	dirPath := filepath.Join(h.uploadPath, "tenants", tenantID, subdir)

	// Create directory if not exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Full file path
	fullPath := filepath.Join(dirPath, filename)

	// Save file
	if err := c.SaveFile(file, fullPath); err != nil {
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative URL path
	relativePath := filepath.Join("/uploads/tenants", tenantID, subdir, filename)

	h.logger.WithFields(logger.Fields{
		"tenant_id": tenantID,
		"file_path": relativePath,
		"file_size": file.Size,
	}).Info("File uploaded successfully")

	return relativePath, fullPath, nil
}
