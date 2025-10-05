package template

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrTemplateNotFound is returned when no template exists for the requested plan
	ErrTemplateNotFound = errors.New("schema template not found")
)

// Repository defines operations for retrieving schema templates
// Templates are plain SQL statements executed after schema provisioning
// to materialise module tables per subscription plan.
type Repository interface {
	GetTemplateByPlan(ctx context.Context, plan string) (string, error)
}

// FileSystemRepository loads templates from the filesystem.
type FileSystemRepository struct {
	basePath string
}

// NewFileSystemRepository creates a new template repository rooted at basePath.
func NewFileSystemRepository(basePath string) *FileSystemRepository {
	return &FileSystemRepository{basePath: basePath}
}

// GetTemplateByPlan returns the SQL template for a subscription plan.
// Fallback order: plan.sql -> default.sql. When no file exists, ErrTemplateNotFound is returned.
func (r *FileSystemRepository) GetTemplateByPlan(ctx context.Context, plan string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	sanitized := strings.ToLower(strings.TrimSpace(plan))
	if sanitized == "" {
		sanitized = "default"
	}

	candidates := []string{
		fmt.Sprintf("%s.sql", sanitized),
		"default.sql",
	}

	for _, candidate := range candidates {
		fullPath := filepath.Join(r.basePath, candidate)
		if _, err := os.Stat(fullPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("failed to stat template %s: %w", fullPath, err)
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to read template %s: %w", fullPath, err)
		}

		return string(content), nil
	}

	return "", ErrTemplateNotFound
}
