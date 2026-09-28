package website

import "github.com/canakyuz/keystone/pkg/clienterr"

// Domain-specific errors for website operations
var (
	// Validation errors
	ErrInvalidWebsiteID   = clienterr.New("invalid website ID")
	ErrTenantIDRequired   = clienterr.New("tenant ID is required")
	ErrNameRequired       = clienterr.New("website name is required")
	ErrInvalidName        = clienterr.New("website name must be between 2 and 100 characters")
	ErrSlugRequired       = clienterr.New("website slug is required")
	ErrInvalidSlug        = clienterr.New("website slug must be between 2 and 50 characters")
	ErrInvalidWebsiteType = clienterr.New("invalid website type")
	ErrInvalidStatus      = clienterr.New("invalid website status")

	// Business logic errors
	ErrNotFound              = clienterr.New("website not found")
	ErrWebsiteExists         = clienterr.New("website already exists")
	ErrSlugExists            = clienterr.New("website slug already exists")
	ErrAlreadyPublished      = clienterr.New("website is already published")
	ErrNotPublished          = clienterr.New("website is not published")
	ErrAlreadyArchived       = clienterr.New("website is already archived")
	ErrNotArchived           = clienterr.New("website is not archived")
	ErrCannotPublish         = clienterr.New("website cannot be published: missing homepage or invalid status")
	ErrCannotPublishArchived = clienterr.New("cannot publish archived website")

	// Custom domain errors
	ErrInvalidCustomDomain         = clienterr.New("invalid custom domain")
	ErrCustomDomainNotSet          = clienterr.New("custom domain is not set")
	ErrCustomDomainAlreadyVerified = clienterr.New("custom domain is already verified")
	ErrCustomDomainNotVerified     = clienterr.New("custom domain is not verified")
	ErrCustomDomainTaken           = clienterr.New("custom domain is already in use")

	// Permission errors
	ErrUnauthorized      = clienterr.New("unauthorized access to website")
	ErrCrossTenantAccess = clienterr.New("cross-tenant website access is not allowed")
	ErrInvalidInput      = clienterr.New("invalid input")
)
