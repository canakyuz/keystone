package website

import (
	"context"

	"github.com/google/uuid"
	"nexspaces-api/internal/domain/website"
)

// Repository defines the interface for website data access
type Repository interface {
	// List returns all websites for a tenant
	List(ctx context.Context, tenantID uuid.UUID) ([]*website.Website, error)

	// Create creates a new website
	Create(ctx context.Context, site *website.Website) error

	// GetByID retrieves a website by ID (tenant-scoped)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*website.Website, error)

	// GetBySlug retrieves a website by slug (global, for public access)
	GetBySlug(ctx context.Context, slug string) (*website.Website, error)

	// Update updates an existing website
	Update(ctx context.Context, site *website.Website) error

	// Delete soft-deletes a website (tenant-scoped)
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// SlugExists checks if a slug already exists
	SlugExists(ctx context.Context, slug string) (bool, error)
}
