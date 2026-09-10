package website

import "github.com/google/uuid"

// CreateWebsiteRequest represents the request to create a website
type CreateWebsiteRequest struct {
	TenantID    uuid.UUID `json:"tenant_id" validate:"required"`
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Slug        string    `json:"slug" validate:"required,min=1,max=255,alphanum"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// UpdateWebsiteRequest represents the request to update a website
type UpdateWebsiteRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// ListWebsitesRequest represents the request to list websites
type ListWebsitesRequest struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
}

// GetWebsiteRequest represents the request to get a website by ID
type GetWebsiteRequest struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
}

// PublishWebsiteRequest represents the request to publish a website
type PublishWebsiteRequest struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
}

// ArchiveWebsiteRequest represents the request to archive a website
type ArchiveWebsiteRequest struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
}

// DeleteWebsiteRequest represents the request to delete a website
type DeleteWebsiteRequest struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
}
