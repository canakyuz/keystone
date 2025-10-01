package project

import "errors"

// Project errors
var (
	// Validation errors
	ErrInvalidProjectID          = errors.New("invalid project ID")
	ErrTenantIDRequired          = errors.New("tenant ID is required")
	ErrTitleRequired             = errors.New("project title is required")
	ErrInvalidTitle              = errors.New("project title must be between 3 and 200 characters")
	ErrSlugRequired              = errors.New("project slug is required")
	ErrInvalidSlug               = errors.New("project slug must be between 3 and 100 characters")
	ErrInvalidCategory           = errors.New("invalid project category")
	ErrInvalidStatus             = errors.New("invalid project status")

	// Business logic errors
	ErrProjectNotFound           = errors.New("project not found")
	ErrProjectAlreadyExists      = errors.New("project with this slug already exists")
	ErrProjectAlreadyCompleted   = errors.New("project is already completed")
	ErrProjectAlreadyInProgress  = errors.New("project is already in progress")
	ErrProjectAlreadyCancelled   = errors.New("project is already cancelled")

	// Multi-tenant errors
	ErrCrossTenantAccess         = errors.New("cross-tenant access is not allowed")
	ErrTenantMismatch            = errors.New("tenant ID mismatch")

	// Permission errors
	ErrUnauthorized              = errors.New("unauthorized access")
	ErrInsufficientPermissions   = errors.New("insufficient permissions")
)
