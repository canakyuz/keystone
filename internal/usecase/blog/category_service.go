package blog

import (
	"context"
	"time"

	"github.com/google/uuid"
	"nexpaces-api/internal/domain/blog"
	"nexpaces-api/pkg/logger"
)

type CategoryService struct {
	repo   blog.CategoryRepository
	logger logger.Logger
}

func NewCategoryService(repo blog.CategoryRepository, logger logger.Logger) *CategoryService {
	return &CategoryService{
		repo:   repo,
		logger: logger,
	}
}

func (s *CategoryService) Create(ctx context.Context, tenantID string, req CreateCategoryRequest) (*CategoryResponse, error) {
	var description string
	if req.Description != nil {
		description = *req.Description
	}

	category := &blog.Category{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: description,
		PostCount:   0,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Metadata == nil {
		category.Metadata = make(map[string]interface{})
	}

	if err := s.repo.Create(ctx, category); err != nil {
		s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err.Error()}).Error("Failed to create category")
		return nil, err
	}

	s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "category_id": category.ID}).Info("Category created successfully")
	resp := ToCategoryResponse(category)
	return &resp, nil
}

func (s *CategoryService) GetByID(ctx context.Context, id string) (*CategoryResponse, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := ToCategoryResponse(category)
	return &resp, nil
}

func (s *CategoryService) GetBySlug(ctx context.Context, slug string) (*CategoryResponse, error) {
	category, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	resp := ToCategoryResponse(category)
	return &resp, nil
}

func (s *CategoryService) List(ctx context.Context, tenantID string, page, limit int) (*CategoryListResponse, error) {
	offset := (page - 1) * limit

	categories, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err.Error()}).Error("Failed to list categories")
		return nil, err
	}

	data := make([]CategoryResponse, len(categories))
	for i, category := range categories {
		data[i] = ToCategoryResponse(category)
	}

	return &CategoryListResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *CategoryService) Update(ctx context.Context, id string, req UpdateCategoryRequest) (*CategoryResponse, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var description string
	if req.Description != nil {
		description = *req.Description
	}

	category.Name = req.Name
	category.Slug = req.Slug
	category.Description = description
	category.Metadata = req.Metadata
	category.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, category); err != nil {
		s.logger.WithFields(logger.Fields{"category_id": id, "error": err.Error()}).Error("Failed to update category")
		return nil, err
	}

	s.logger.WithFields(logger.Fields{"category_id": id}).Info("Category updated successfully")
	resp := ToCategoryResponse(category)
	return &resp, nil
}

func (s *CategoryService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{"category_id": id, "error": err.Error()}).Error("Failed to delete category")
		return err
	}

	s.logger.WithFields(logger.Fields{"category_id": id}).Info("Category deleted successfully")
	return nil
}
