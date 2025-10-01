package website

import "errors"

// Domain-specific errors for website operations
var (
	// Validation errors
	ErrInvalidWebsiteID   = errors.New("invalid website ID")
	ErrTenantIDRequired   = errors.New("tenant ID is required")
	ErrNameRequired       = errors.New("website name is required")
	ErrInvalidName        = errors.New("website name must be between 2 and 100 characters")
	ErrSlugRequired       = errors.New("website slug is required")
	ErrInvalidSlug        = errors.New("website slug must be between 2 and 50 characters")
	ErrInvalidWebsiteType = errors.New("invalid website type")
	ErrInvalidStatus      = errors.New("invalid website status")

	// Business logic errors
	ErrNotFound          = errors.New("website not found")
	ErrWebsiteExists     = errors.New("website already exists")
	ErrSlugExists        = errors.New("website slug already exists")
	ErrAlreadyPublished  = errors.New("website is already published")
	ErrNotPublished      = errors.New("website is not published")
	ErrAlreadyArchived   = errors.New("website is already archived")
	ErrNotArchived       = errors.New("website is not archived")
	ErrCannotPublish     = errors.New("website cannot be published: missing homepage or invalid status")
	ErrCannotPublishArchived = errors.New("cannot publish archived website")

	// Custom domain errors
	ErrInvalidCustomDomain         = errors.New("invalid custom domain")
	ErrCustomDomainNotSet          = errors.New("custom domain is not set")
	ErrCustomDomainAlreadyVerified = errors.New("custom domain is already verified")
	ErrCustomDomainNotVerified     = errors.New("custom domain is not verified")
	ErrCustomDomainTaken           = errors.New("custom domain is already in use")

	// Permission errors
	ErrUnauthorized      = errors.New("unauthorized access to website")
	ErrCrossTenantAccess = errors.New("cross-tenant website access is not allowed")
	ErrInvalidInput      = errors.New("invalid input")
)
