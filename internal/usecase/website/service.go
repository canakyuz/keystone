package website

import (
	"context"
	"strings"

	"github.com/canakyuz/keystone/internal/domain/website"
	websiteRepo "github.com/canakyuz/keystone/internal/repository/website"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Service handles website business logic
type Service struct {
	repo      websiteRepo.Repository
	validator *validator.Validate
}

// NewService creates a new website service
func NewService(repo websiteRepo.Repository) *Service {
	return &Service{
		repo:      repo,
		validator: validator.New(),
	}
}

// List retrieves all websites for a tenant
func (s *Service) List(ctx context.Context, req ListWebsitesRequest) ([]*website.Website, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, website.ErrInvalidInput
	}

	return s.repo.List(ctx, req.TenantID)
}

// Create creates a new website
func (s *Service) Create(ctx context.Context, req CreateWebsiteRequest) (*website.Website, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, website.ErrInvalidInput
	}

	// Normalize slug
	slug := normalizeSlug(req.Slug)

	// Check if slug exists
	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, website.ErrSlugExists
	}

	// Create website entity
	site := &website.Website{
		ID:          uuid.New(),
		TenantID:    req.TenantID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Status:      website.StatusDraft,
	}

	// Persist
	if err := s.repo.Create(ctx, site); err != nil {
		return nil, err
	}

	return site, nil
}

// GetByID retrieves a website by ID (tenant-scoped)
func (s *Service) GetByID(ctx context.Context, req GetWebsiteRequest) (*website.Website, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, website.ErrInvalidInput
	}

	return s.repo.GetByID(ctx, req.ID, req.TenantID)
}

// GetBySlug retrieves a website by slug (public, no tenant check)
func (s *Service) GetBySlug(ctx context.Context, slug string) (*website.Website, error) {
	if slug == "" {
		return nil, website.ErrInvalidInput
	}

	return s.repo.GetBySlug(ctx, normalizeSlug(slug))
}

// Update updates a website
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, req UpdateWebsiteRequest) (*website.Website, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, website.ErrInvalidInput
	}

	// Get existing website
	site, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Update fields
	site.Update(req.Name, req.Description)

	// Persist
	if err := s.repo.Update(ctx, site); err != nil {
		return nil, err
	}

	return site, nil
}

// Publish publishes a website
func (s *Service) Publish(ctx context.Context, req PublishWebsiteRequest) (*website.Website, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, website.ErrInvalidInput
	}

	// Get existing website
	site, err := s.repo.GetByID(ctx, req.ID, req.TenantID)
	if err != nil {
		return nil, err
	}

	// Publish (domain logic)
	if err := site.Publish(); err != nil {
		return nil, err
	}

	// Persist
	if err := s.repo.Update(ctx, site); err != nil {
		return nil, err
	}

	return site, nil
}

// Archive archives a website
func (s *Service) Archive(ctx context.Context, req ArchiveWebsiteRequest) (*website.Website, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, website.ErrInvalidInput
	}

	// Get existing website
	site, err := s.repo.GetByID(ctx, req.ID, req.TenantID)
	if err != nil {
		return nil, err
	}

	// Archive (domain logic)
	if err := site.Archive(); err != nil {
		return nil, err
	}

	// Persist
	if err := s.repo.Update(ctx, site); err != nil {
		return nil, err
	}

	return site, nil
}

// Delete soft-deletes a website
func (s *Service) Delete(ctx context.Context, req DeleteWebsiteRequest) error {
	if err := s.validator.Struct(req); err != nil {
		return website.ErrInvalidInput
	}

	return s.repo.Delete(ctx, req.ID, req.TenantID)
}

// normalizeSlug normalizes a slug (lowercase, trim spaces)
func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}
