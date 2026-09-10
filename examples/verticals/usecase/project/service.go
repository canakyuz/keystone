package project

import (
	"context"
	"fmt"

	"github.com/canakyuz/keystone/examples/verticals/domain/project"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/validator"
)

// Service implements project business logic
type Service struct {
	repo      project.Repository
	validator *validator.Validator
	logger    *logger.Logger
}

// NewService creates a new project service
func NewService(repo project.Repository, validator *validator.Validator, logger *logger.Logger) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

// Create creates a new project
func (s *Service) Create(ctx context.Context, tenantID string, req *CreateProjectRequest) (*ProjectResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Check if slug already exists
	exists, err := s.repo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		s.logger.WithFields(logger.Fields{"slug": req.Slug, "error": err}).Error("failed to check project slug existence")
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}
	if exists {
		return nil, project.ErrProjectAlreadyExists
	}

	// Create project entity
	p, err := project.New(
		tenantID,
		req.Title,
		req.Slug,
		req.Description,
		project.ProjectCategory(req.Category),
		project.ProjectStatus(req.Status),
	)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	p.Content = req.Content
	p.Client = req.Client
	p.Technologies = req.Technologies
	p.CoverImage = req.CoverImage
	p.LiveURL = req.LiveURL
	p.GithubURL = req.GithubURL
	p.StartDate = req.StartDate
	p.EndDate = req.EndDate

	// Save to repository
	if err := s.repo.Create(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err}).Error("failed to create project")
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.WithFields(logger.Fields{"project_id": p.ID, "tenant_id": tenantID}).Info("project created successfully")

	return toProjectResponse(p), nil
}

// GetByID retrieves a project by ID
func (s *Service) GetByID(ctx context.Context, id string) (*ProjectResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toProjectResponse(p), nil
}

// GetBySlug retrieves a project by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*ProjectResponse, error) {
	p, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Increment view count
	if err := s.repo.IncrementViewCount(ctx, p.ID); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": p.ID, "error": err}).Warn("failed to increment view count")
	}

	return toProjectResponse(p), nil
}

// List retrieves projects with filters
func (s *Service) List(ctx context.Context, req *ListProjectsRequest) (*ListProjectsResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// Build filters
	filters := project.ListFilters{
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PageSize,
		Offset:    (req.Page - 1) * req.PageSize,
	}

	if req.Status != nil {
		status := project.ProjectStatus(*req.Status)
		filters.Status = &status
	}

	if req.Category != nil {
		category := project.ProjectCategory(*req.Category)
		filters.Category = &category
	}

	if req.Featured != nil {
		filters.Featured = req.Featured
	}

	// Get projects
	projects, total, err := s.repo.List(ctx, filters)
	if err != nil {
		s.logger.WithFields(logger.Fields{"error": err}).Error("failed to list projects")
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Convert to response
	projectResponses := make([]*ProjectResponse, len(projects))
	for i, p := range projects {
		projectResponses[i] = toProjectResponse(p)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ListProjectsResponse{
		Projects:   projectResponses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing project
func (s *Service) Update(ctx context.Context, id string, req *UpdateProjectRequest) (*ProjectResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Get existing project
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Title != nil {
		p.Title = *req.Title
	}
	if req.Slug != nil {
		// Check if new slug already exists
		if *req.Slug != p.Slug {
			exists, err := s.repo.ExistsBySlug(ctx, *req.Slug)
			if err != nil {
				return nil, fmt.Errorf("failed to check slug existence: %w", err)
			}
			if exists {
				return nil, project.ErrProjectAlreadyExists
			}
			p.Slug = *req.Slug
		}
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Content != nil {
		p.Content = *req.Content
	}
	if req.Category != nil {
		p.Category = project.ProjectCategory(*req.Category)
	}
	if req.Status != nil {
		p.Status = project.ProjectStatus(*req.Status)
	}
	if req.Client != nil {
		p.Client = *req.Client
	}
	if req.Technologies != nil {
		p.Technologies = req.Technologies
	}
	if req.CoverImage != nil {
		p.CoverImage = *req.CoverImage
	}
	if req.LiveURL != nil {
		p.LiveURL = *req.LiveURL
	}
	if req.GithubURL != nil {
		p.GithubURL = *req.GithubURL
	}
	if req.StartDate != nil {
		p.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		p.EndDate = req.EndDate
	}
	if req.Featured != nil {
		p.Featured = *req.Featured
	}
	if req.SortOrder != nil {
		p.SortOrder = *req.SortOrder
	}

	// Validate updated project
	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to update project")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.WithFields(logger.Fields{"project_id": id}).Info("project updated successfully")

	return toProjectResponse(p), nil
}

// Delete soft deletes a project
func (s *Service) Delete(ctx context.Context, id string) error {
	// Check if project exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete project
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to delete project")
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.WithFields(logger.Fields{"project_id": id}).Info("project deleted successfully")

	return nil
}

// MarkAsCompleted marks project as completed
func (s *Service) MarkAsCompleted(ctx context.Context, id string) (*ProjectResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := p.MarkAsCompleted(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to mark project as completed")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.WithFields(logger.Fields{"project_id": id}).Info("project marked as completed")

	return toProjectResponse(p), nil
}

// SetFeatured sets or unsets project as featured
func (s *Service) SetFeatured(ctx context.Context, id string, featured bool) (*ProjectResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	p.SetFeatured(featured)

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to set project featured status")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.WithFields(logger.Fields{"project_id": id, "featured": featured}).Info("project featured status updated")

	return toProjectResponse(p), nil
}

// GetFeatured retrieves featured projects
func (s *Service) GetFeatured(ctx context.Context, limit int) ([]*ProjectResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	projects, err := s.repo.GetFeatured(ctx, limit)
	if err != nil {
		s.logger.WithFields(logger.Fields{"error": err}).Error("failed to get featured projects")
		return nil, fmt.Errorf("failed to get featured projects: %w", err)
	}

	responses := make([]*ProjectResponse, len(projects))
	for i, p := range projects {
		responses[i] = toProjectResponse(p)
	}

	return responses, nil
}

// GetByCategory retrieves projects by category
func (s *Service) GetByCategory(ctx context.Context, category string, page, pageSize int) (*ListProjectsResponse, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	projects, total, err := s.repo.GetByCategory(ctx, project.ProjectCategory(category), pageSize, offset)
	if err != nil {
		s.logger.WithFields(logger.Fields{"category": category, "error": err}).Error("failed to get projects by category")
		return nil, fmt.Errorf("failed to get projects by category: %w", err)
	}

	responses := make([]*ProjectResponse, len(projects))
	for i, p := range projects {
		responses[i] = toProjectResponse(p)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ListProjectsResponse{
		Projects:   responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetStats returns project statistics
func (s *Service) GetStats(ctx context.Context) (*ProjectStatsResponse, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"error": err}).Error("failed to get project stats")
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &ProjectStatsResponse{
		Total:      stats["total"].(int64),
		Completed:  stats["completed"].(int64),
		InProgress: stats["in_progress"].(int64),
		Planned:    stats["planned"].(int64),
		Featured:   stats["featured"].(int64),
		TotalViews: stats["total_views"].(int64),
	}, nil
}

// AddImage adds an image to project
func (s *Service) AddImage(ctx context.Context, id string, req *AddImageRequest) (*ProjectResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	p.AddImage(req.ImageURL)

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to add image")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return toProjectResponse(p), nil
}

// RemoveImage removes an image from project
func (s *Service) RemoveImage(ctx context.Context, id string, req *RemoveImageRequest) (*ProjectResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	p.RemoveImage(req.ImageURL)

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to remove image")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return toProjectResponse(p), nil
}

// AddTechnology adds a technology to project
func (s *Service) AddTechnology(ctx context.Context, id string, req *AddTechnologyRequest) (*ProjectResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	p.AddTechnology(req.Technology)

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to add technology")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return toProjectResponse(p), nil
}

// RemoveTechnology removes a technology from project
func (s *Service) RemoveTechnology(ctx context.Context, id string, req *RemoveTechnologyRequest) (*ProjectResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	p.RemoveTechnology(req.Technology)

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.WithFields(logger.Fields{"project_id": id, "error": err}).Error("failed to remove technology")
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return toProjectResponse(p), nil
}

// toProjectResponse converts domain entity to response DTO
func toProjectResponse(p *project.Project) *ProjectResponse {
	return &ProjectResponse{
		ID:           p.ID,
		TenantID:     p.TenantID,
		Title:        p.Title,
		Slug:         p.Slug,
		Description:  p.Description,
		Content:      p.Content,
		Category:     string(p.Category),
		Status:       string(p.Status),
		Client:       p.Client,
		Technologies: p.Technologies,
		Images:       p.Images,
		CoverImage:   p.CoverImage,
		LiveURL:      p.LiveURL,
		GithubURL:    p.GithubURL,
		StartDate:    p.StartDate,
		EndDate:      p.EndDate,
		Featured:     p.Featured,
		SortOrder:    p.SortOrder,
		ViewCount:    p.ViewCount,
		Metadata:     p.Metadata,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
