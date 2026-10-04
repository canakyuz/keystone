package template

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// The templates are compiled into the binary. They used to be read from a templates/
// directory relative to the working directory, which the container image never copied,
// so in a container every plan would silently have fallen through to an empty schema.
//
//go:embed sql/*.sql
var embedded embed.FS

var (
	// ErrTemplateNotFound is returned when no template exists for the requested plan
	ErrTemplateNotFound = errors.New("schema template not found")
)

// EmbeddedRepository serves the templates compiled into the binary.
type EmbeddedRepository struct{}

// NewRepository returns a repository over the templates embedded in the binary.
func NewRepository() *EmbeddedRepository {
	return &EmbeddedRepository{}
}

// GetTemplateByPlan returns the SQL template for a subscription plan.
// Fallback order: plan.sql -> default.sql. When no file exists, ErrTemplateNotFound is returned.
func (r *EmbeddedRepository) GetTemplateByPlan(ctx context.Context, plan string) (string, error) {
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
		content, err := fs.ReadFile(embedded, "sql/"+candidate)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to read template %s: %w", candidate, err)
		}
		return string(content), nil
	}

	return "", ErrTemplateNotFound
}
