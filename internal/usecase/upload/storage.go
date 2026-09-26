package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/google/uuid"

	uploadrepo "github.com/canakyuz/keystone/internal/repository/upload"
	"github.com/canakyuz/keystone/pkg/logger"
)

// BasePath is where accepted files are stored, and what the sweeper walks.
const BasePath = "./uploads"

// Size limits, in bytes.
//
// Each has to fit inside the request body limit (MAX_REQUEST_SIZE, 4MB by default) with
// room for the multipart framing around the file. The general limit used to be 10MB, which
// no request could reach: the body was refused before the handler ran.
const (
	MaxLogoSize    = 2 * 1024 * 1024
	MaxFaviconSize = 500 * 1024
	MaxFileSize    = 3 * 1024 * 1024
)

// Category is a kind of upload: where it is stored, how large it may be and which types
// it may be stored as.
//
// The types are the ones read from the file's own first bytes, never the one the client
// declares. SVG is not among them. An SVG is a document that can carry script, served from
// this origin it runs with this origin's authority, and no byte signature tells a harmless
// one from a hostile one.
type Category struct {
	Name    string
	Subdir  string
	MaxSize int64
	Types   []string
}

// The categories the API accepts.
var (
	Logo    = Category{"logo", "logo", MaxLogoSize, []string{"image/png", "image/jpeg", "image/webp"}}
	Favicon = Category{"favicon", "favicon", MaxFaviconSize, []string{"image/x-icon", "image/png"}}
	Image   = Category{"image", "images", MaxFileSize, []string{"image/png", "image/jpeg", "image/gif", "image/webp"}}
)

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

// RejectedError is a file the category does not accept. Its message is safe to return to
// the client.
type RejectedError struct{ Reason string }

// Error returns the reason, which is safe to show the client.
func (e *RejectedError) Error() string { return e.Reason }

// Recorder records a stored file.
type Recorder interface {
	Create(ctx context.Context, record *uploadrepo.Record) error
}

// Storage accepts files onto disk and records them.
type Storage struct {
	log      *logger.Logger
	records  Recorder
	basePath string
}

// NewStorage creates the storage rooted at BasePath.
func NewStorage(log *logger.Logger, records Recorder) *Storage {
	return &Storage{log: log, records: records, basePath: BasePath}
}

// Accept validates file for category, writes it under the tenant's directory and records
// it. The returned record carries the path the file is served from.
func (s *Storage) Accept(ctx context.Context, tenantID string, category Category, file *multipart.FileHeader) (*uploadrepo.Record, error) {
	contentType, err := detect(file, category)
	if err != nil {
		return nil, err
	}

	name := uuid.New().String() + extensions[contentType]
	dir := filepath.Join(s.basePath, "tenants", tenantID, category.Subdir)
	diskPath := filepath.Join(dir, name)

	if err := write(file, dir, diskPath); err != nil {
		return nil, err
	}

	record := &uploadrepo.Record{
		TenantID:    tenantID,
		Category:    category.Name,
		Path:        path.Join("/uploads/tenants", tenantID, category.Subdir, name),
		ContentType: contentType,
		SizeBytes:   file.Size,
	}
	if err := s.records.Create(ctx, record); err != nil {
		// The file is on disk with no record of it. Removing it keeps the two in step: a
		// stored file nobody can account for is what the record exists to rule out.
		if removeErr := os.Remove(diskPath); removeErr != nil {
			s.log.ErrorWithErr(removeErr, "failed to remove an unrecorded upload")
		}
		return nil, err
	}

	s.log.WithFields(logger.Fields{
		"tenant_id": tenantID,
		"file_path": record.Path,
		"file_size": file.Size,
	}).Info("File uploaded successfully")

	return record, nil
}

// detect checks the file's size and reads its type from its first bytes, returning that
// type when the category accepts it.
func detect(file *multipart.FileHeader, category Category) (string, error) {
	if file.Size > category.MaxSize {
		return "", &RejectedError{Reason: fmt.Sprintf("file size exceeds limit (%d bytes)", category.MaxSize)}
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
	if !slices.Contains(category.Types, contentType) {
		return "", &RejectedError{Reason: fmt.Sprintf("file type not allowed (allowed: %v)", category.Types)}
	}

	return contentType, nil
}

func write(file *multipart.FileHeader, dir, diskPath string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(diskPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(diskPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return dst.Close()
}
