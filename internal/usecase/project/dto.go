package project

import "time"

// CreateProjectRequest represents request to create a project
type CreateProjectRequest struct {
	Title        string     `json:"title" validate:"required,min=3,max=200"`
	Slug         string     `json:"slug" validate:"required,min=3,max=100"`
	Description  string     `json:"description" validate:"required"`
	Content      string     `json:"content,omitempty"`
	Category     string     `json:"category" validate:"required"`
	Status       string     `json:"status" validate:"required"`
	Client       string     `json:"client,omitempty"`
	Technologies []string   `json:"technologies,omitempty"`
	CoverImage   string     `json:"cover_image,omitempty"`
	LiveURL      string     `json:"live_url,omitempty"`
	GithubURL    string     `json:"github_url,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
}

// UpdateProjectRequest represents request to update a project
type UpdateProjectRequest struct {
	Title        *string    `json:"title,omitempty" validate:"omitempty,min=3,max=200"`
	Slug         *string    `json:"slug,omitempty" validate:"omitempty,min=3,max=100"`
	Description  *string    `json:"description,omitempty"`
	Content      *string    `json:"content,omitempty"`
	Category     *string    `json:"category,omitempty"`
	Status       *string    `json:"status,omitempty"`
	Client       *string    `json:"client,omitempty"`
	Technologies []string   `json:"technologies,omitempty"`
	CoverImage   *string    `json:"cover_image,omitempty"`
	LiveURL      *string    `json:"live_url,omitempty"`
	GithubURL    *string    `json:"github_url,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Featured     *bool      `json:"featured,omitempty"`
	SortOrder    *int       `json:"sort_order,omitempty"`
}

// ListProjectsRequest represents request to list projects
type ListProjectsRequest struct {
	Status    *string `json:"status,omitempty"`
	Category  *string `json:"category,omitempty"`
	Featured  *bool   `json:"featured,omitempty"`
	Search    string  `json:"search,omitempty"`
	SortBy    string  `json:"sort_by,omitempty" validate:"omitempty,oneof=created_at updated_at title view_count"`
	SortOrder string  `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
	Page      int     `json:"page,omitempty" validate:"omitempty,min=1"`
	PageSize  int     `json:"page_size,omitempty" validate:"omitempty,min=1,max=100"`
}

// ProjectResponse represents project response
type ProjectResponse struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Title        string                 `json:"title"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	Content      string                 `json:"content,omitempty"`
	Category     string                 `json:"category"`
	Status       string                 `json:"status"`
	Client       string                 `json:"client,omitempty"`
	Technologies []string               `json:"technologies,omitempty"`
	Images       []string               `json:"images,omitempty"`
	CoverImage   string                 `json:"cover_image,omitempty"`
	LiveURL      string                 `json:"live_url,omitempty"`
	GithubURL    string                 `json:"github_url,omitempty"`
	StartDate    *time.Time             `json:"start_date,omitempty"`
	EndDate      *time.Time             `json:"end_date,omitempty"`
	Featured     bool                   `json:"featured"`
	SortOrder    int                    `json:"sort_order"`
	ViewCount    int64                  `json:"view_count"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ListProjectsResponse represents paginated project list response
type ListProjectsResponse struct {
	Projects   []*ProjectResponse `json:"projects"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ProjectStatsResponse represents project statistics
type ProjectStatsResponse struct {
	Total      int64 `json:"total"`
	Completed  int64 `json:"completed"`
	InProgress int64 `json:"in_progress"`
	Planned    int64 `json:"planned"`
	Featured   int64 `json:"featured"`
	TotalViews int64 `json:"total_views"`
}

// AddImageRequest represents request to add image
type AddImageRequest struct {
	ImageURL string `json:"image_url" validate:"required,url"`
}

// RemoveImageRequest represents request to remove image
type RemoveImageRequest struct {
	ImageURL string `json:"image_url" validate:"required"`
}

// AddTechnologyRequest represents request to add technology
type AddTechnologyRequest struct {
	Technology string `json:"technology" validate:"required,min=1,max=50"`
}

// RemoveTechnologyRequest represents request to remove technology
type RemoveTechnologyRequest struct {
	Technology string `json:"technology" validate:"required"`
}
