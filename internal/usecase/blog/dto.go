package blog

import (
	"time"

	"nexpaces-api/internal/domain/blog"
)

// Post DTOs
type CreatePostRequest struct {
	CategoryID  *string   `json:"category_id,omitempty"`
	Title       string    `json:"title" validate:"required"`
	Slug        string    `json:"slug" validate:"required"`
	Content     *string   `json:"content,omitempty"`
	Excerpt     *string   `json:"excerpt,omitempty"`
	Status      string    `json:"status" validate:"required,oneof=draft published archived"`
	Featured    bool      `json:"featured"`
	Image       *string   `json:"image,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdatePostRequest struct {
	CategoryID  *string   `json:"category_id,omitempty"`
	Title       string    `json:"title" validate:"required"`
	Slug        string    `json:"slug" validate:"required"`
	Content     *string   `json:"content,omitempty"`
	Excerpt     *string   `json:"excerpt,omitempty"`
	Status      string    `json:"status" validate:"required,oneof=draft published archived"`
	Featured    bool      `json:"featured"`
	Image       *string   `json:"image,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PostResponse struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	CategoryID  *string                `json:"category_id,omitempty"`
	Title       string                 `json:"title"`
	Slug        string                 `json:"slug"`
	Content     *string                `json:"content,omitempty"`
	Excerpt     *string                `json:"excerpt,omitempty"`
	Status      string                 `json:"status"`
	Featured    bool                   `json:"featured"`
	ViewCount   int64                  `json:"view_count"`
	Image       *string                `json:"image,omitempty"`
	Tags        []string               `json:"tags"`
	PublishedAt *time.Time             `json:"published_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   *string                `json:"created_by,omitempty"`
	UpdatedBy   *string                `json:"updated_by,omitempty"`
}

type PostListResponse struct {
	Data  []PostResponse `json:"data"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// Category DTOs
type CreateCategoryRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Slug        string                 `json:"slug" validate:"required"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateCategoryRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Slug        string                 `json:"slug" validate:"required"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CategoryResponse struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description *string                `json:"description,omitempty"`
	PostCount   int                    `json:"post_count"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type CategoryListResponse struct {
	Data  []CategoryResponse `json:"data"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// Mappers
func ToPostResponse(post *blog.Post) PostResponse {
	var categoryID, content, excerpt, image, createdBy, updatedBy *string
	if post.CategoryID != "" {
		categoryID = &post.CategoryID
	}
	if post.Content != "" {
		content = &post.Content
	}
	if post.Excerpt != "" {
		excerpt = &post.Excerpt
	}
	if post.Image != "" {
		image = &post.Image
	}
	if post.CreatedBy != "" {
		createdBy = &post.CreatedBy
	}
	if post.UpdatedBy != "" {
		updatedBy = &post.UpdatedBy
	}

	return PostResponse{
		ID:          post.ID,
		TenantID:    post.TenantID,
		CategoryID:  categoryID,
		Title:       post.Title,
		Slug:        post.Slug,
		Content:     content,
		Excerpt:     excerpt,
		Status:      string(post.Status),
		Featured:    post.Featured,
		ViewCount:   post.ViewCount,
		Image:       image,
		Tags:        post.Tags,
		PublishedAt: post.PublishedAt,
		Metadata:    post.Metadata,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		CreatedBy:   createdBy,
		UpdatedBy:   updatedBy,
	}
}

func ToCategoryResponse(category *blog.Category) CategoryResponse {
	var description *string
	if category.Description != "" {
		description = &category.Description
	}

	return CategoryResponse{
		ID:          category.ID,
		TenantID:    category.TenantID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: description,
		PostCount:   category.PostCount,
		Metadata:    category.Metadata,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}
