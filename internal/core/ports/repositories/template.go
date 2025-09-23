package repositories

import (
	"context"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/template"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// TemplateRepository defines the interface for template data access
type TemplateRepository interface {
	// Create creates a new template
	Create(ctx context.Context, template *template.Template) error

	// GetByID retrieves a template by ID
	GetByID(ctx context.Context, id template.TemplateID) (*template.Template, error)

	// GetBySlug retrieves a template by slug
	GetBySlug(ctx context.Context, slug string) (*template.Template, error)

	// Update updates an existing template
	Update(ctx context.Context, template *template.Template) error

	// Delete deletes a template (soft delete)
	Delete(ctx context.Context, id template.TemplateID) error

	// List retrieves templates with filtering and pagination
	List(ctx context.Context, filter TemplateFilter) (*TemplateList, error)

	// GetPublicTemplates retrieves public templates for marketplace
	GetPublicTemplates(ctx context.Context, filter PublicTemplateFilter) (*TemplateList, error)

	// GetTenantTemplates retrieves templates for a specific tenant
	GetTenantTemplates(ctx context.Context, tenantID tenant.TenantID, filter TemplateFilter) (*TemplateList, error)

	// GetByAuthor retrieves templates by author
	GetByAuthor(ctx context.Context, authorID user.UserID, filter TemplateFilter) (*TemplateList, error)

	// GetByCategory retrieves templates by category
	GetByCategory(ctx context.Context, category template.Category, filter TemplateFilter) (*TemplateList, error)

	// Search searches templates by text
	Search(ctx context.Context, query string, filter TemplateFilter) (*TemplateList, error)

	// ExistsBySlug checks if a template with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// Count returns the total number of templates matching the filter
	Count(ctx context.Context, filter TemplateFilter) (int64, error)

	// GetFeatured retrieves featured templates
	GetFeatured(ctx context.Context, limit int) ([]*template.Template, error)

	// GetPopular retrieves popular templates based on downloads/installs
	GetPopular(ctx context.Context, limit int) ([]*template.Template, error)

	// GetRecentlyUpdated retrieves recently updated templates
	GetRecentlyUpdated(ctx context.Context, limit int) ([]*template.Template, error)

	// IncrementDownloads increments download count
	IncrementDownloads(ctx context.Context, id template.TemplateID) error

	// IncrementInstalls increments install count
	IncrementInstalls(ctx context.Context, id template.TemplateID) error

	// IncrementViews increments view count
	IncrementViews(ctx context.Context, id template.TemplateID) error

	// UpdateRating updates template rating
	UpdateRating(ctx context.Context, id template.TemplateID, rating float64) error

	// GetRelated retrieves related templates
	GetRelated(ctx context.Context, templateID template.TemplateID, limit int) ([]*template.Template, error)

	// GetDependencies retrieves template dependencies
	GetDependencies(ctx context.Context, templateID template.TemplateID) ([]*TemplateDependency, error)

	// GetInstallations retrieves template installations for tenant
	GetInstallations(ctx context.Context, tenantID tenant.TenantID) ([]*TemplateInstallation, error)

	// GetInstallation retrieves specific installation
	GetInstallation(ctx context.Context, tenantID tenant.TenantID, templateID template.TemplateID) (*TemplateInstallation, error)

	// CreateInstallation creates a new installation record
	CreateInstallation(ctx context.Context, installation *TemplateInstallation) error

	// UpdateInstallation updates installation record
	UpdateInstallation(ctx context.Context, installation *TemplateInstallation) error

	// DeleteInstallation removes installation record
	DeleteInstallation(ctx context.Context, tenantID tenant.TenantID, templateID template.TemplateID) error
}

// TemplateFilter represents filtering options for template queries
type TemplateFilter struct {
	// Visibility filtering
	Visibility *template.Visibility

	// Status filtering
	Status *template.TemplateStatus

	// Category filtering
	Categories []template.Category

	// Author filtering
	AuthorID *user.UserID

	// Tenant filtering (for private templates)
	TenantID *tenant.TenantID

	// Pricing filtering
	PricingModel *template.PricingModel
	IsFree       *bool
	MaxPrice     *int64 // in cents

	// Tags filtering
	Tags    []string // Templates must have ALL these tags
	AnyTags []string // Templates must have ANY of these tags

	// Search term (searches in name, description, tags)
	Search string

	// Created date range
	CreatedAfter  *shared.Timestamp
	CreatedBefore *shared.Timestamp

	// Updated date range
	UpdatedAfter  *shared.Timestamp
	UpdatedBefore *shared.Timestamp

	// Published date range
	PublishedAfter  *shared.Timestamp
	PublishedBefore *shared.Timestamp

	// Statistics filtering
	MinDownloads *int64
	MinRating    *float64

	// Framework/technology filtering
	Framework *string
	Language  *string

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // name, created_at, updated_at, published_at, downloads, rating, installs
	SortOrder string // asc, desc

	// Include related data
	IncludeStatistics bool
	IncludeAuthor     bool
	IncludeTenant     bool
}

// PublicTemplateFilter represents filtering for public marketplace
type PublicTemplateFilter struct {
	// Category filtering
	Categories []template.Category

	// Pricing filtering
	PricingModel *template.PricingModel
	IsFree       *bool
	MaxPrice     *int64

	// Tags filtering
	Tags    []string
	AnyTags []string

	// Search term
	Search string

	// Statistics filtering
	MinDownloads *int64
	MinRating    *float64

	// Framework/technology filtering
	Framework *string
	Language  *string

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // name, created_at, published_at, downloads, rating, installs
	SortOrder string // asc, desc

	// Featured templates only
	FeaturedOnly bool
}

// TemplateList represents a paginated list of templates
type TemplateList struct {
	Items      []*template.Template `json:"items"`
	Total      int64                `json:"total"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	HasMore    bool                 `json:"has_more"`
	TotalPages int                  `json:"total_pages"`
}

// TemplateDependency represents a template dependency
type TemplateDependency struct {
	ID             shared.ID           `json:"id"`
	TemplateID     template.TemplateID `json:"template_id"`
	DependencyID   template.TemplateID `json:"dependency_id"`
	DependencyType DependencyType      `json:"dependency_type"`
	MinVersion     string              `json:"min_version"`
	MaxVersion     *string             `json:"max_version"`
	Required       bool                `json:"required"`
	Description    string              `json:"description"`
	CreatedAt      shared.Timestamp    `json:"created_at"`
}

// DependencyType represents the type of dependency
type DependencyType string

const (
	DependencyTypeTemplate    DependencyType = "template"
	DependencyTypeModule      DependencyType = "module"
	DependencyTypeTheme       DependencyType = "theme"
	DependencyTypeExtension   DependencyType = "extension"
	DependencyTypeIntegration DependencyType = "integration"
)

// TemplateInstallation represents a template installation
type TemplateInstallation struct {
	ID             shared.ID              `json:"id"`
	TenantID       tenant.TenantID        `json:"tenant_id"`
	TemplateID     template.TemplateID    `json:"template_id"`
	InstalledBy    user.UserID            `json:"installed_by"`
	Version        string                 `json:"version"`
	Status         InstallationStatus     `json:"status"`
	Configuration  map[string]interface{} `json:"configuration"`
	CustomSettings map[string]interface{} `json:"custom_settings"`
	Price          *shared.Money          `json:"price"`
	LicenseKey     *string                `json:"license_key"`
	ExpiresAt      *shared.Timestamp      `json:"expires_at"`
	InstalledAt    shared.Timestamp       `json:"installed_at"`
	UpdatedAt      shared.Timestamp       `json:"updated_at"`
	LastUsedAt     *shared.Timestamp      `json:"last_used_at"`
}

// InstallationStatus represents installation status
type InstallationStatus string

const (
	InstallationActive      InstallationStatus = "active"
	InstallationInactive    InstallationStatus = "inactive"
	InstallationUninstalled InstallationStatus = "uninstalled"
	InstallationExpired     InstallationStatus = "expired"
	InstallationError       InstallationStatus = "error"
)

// TemplateReview represents a template review/rating
type TemplateReview struct {
	ID         shared.ID           `json:"id"`
	TemplateID template.TemplateID `json:"template_id"`
	TenantID   tenant.TenantID     `json:"tenant_id"`
	UserID     user.UserID         `json:"user_id"`
	Rating     int                 `json:"rating"` // 1-5 stars
	Title      string              `json:"title"`
	Comment    string              `json:"comment"`
	Verified   bool                `json:"verified"` // Verified purchase
	Helpful    int                 `json:"helpful"`  // Helpful votes
	CreatedAt  shared.Timestamp    `json:"created_at"`
	UpdatedAt  shared.Timestamp    `json:"updated_at"`
}

// TemplateReviewRepository defines the interface for template review data access
type TemplateReviewRepository interface {
	// Create creates a new review
	Create(ctx context.Context, review *TemplateReview) error

	// GetByID retrieves a review by ID
	GetByID(ctx context.Context, id shared.ID) (*TemplateReview, error)

	// GetByTemplate retrieves reviews for a template
	GetByTemplate(ctx context.Context, templateID template.TemplateID, filter ReviewFilter) ([]*TemplateReview, error)

	// GetByUser retrieves reviews by a user
	GetByUser(ctx context.Context, userID user.UserID) ([]*TemplateReview, error)

	// Update updates a review
	Update(ctx context.Context, review *TemplateReview) error

	// Delete deletes a review
	Delete(ctx context.Context, id shared.ID) error

	// GetAverageRating returns average rating for template
	GetAverageRating(ctx context.Context, templateID template.TemplateID) (float64, int, error)

	// IncrementHelpful increments helpful count
	IncrementHelpful(ctx context.Context, id shared.ID) error
}

// ReviewFilter represents filtering options for review queries
type ReviewFilter struct {
	Rating    *int  // Filter by specific rating
	MinRating *int  // Minimum rating
	Verified  *bool // Verified purchases only
	Limit     int
	Offset    int
	SortBy    string // created_at, rating, helpful
	SortOrder string // asc, desc
}

// Validation methods
func (f *TemplateFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 1000 {
		return shared.NewValidationError("limit must be between 0 and 1000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	if f.MinRating != nil && (*f.MinRating < 0 || *f.MinRating > 5) {
		return shared.NewValidationError("rating must be between 0 and 5")
	}

	validSortFields := []string{
		"name", "created_at", "updated_at", "published_at",
		"downloads", "rating", "installs", "views",
	}
	if f.SortBy != "" {
		valid := false
		for _, field := range validSortFields {
			if f.SortBy == field {
				valid = true
				break
			}
		}
		if !valid {
			return shared.NewValidationError("invalid sort field")
		}
	}

	if f.SortOrder != "" && f.SortOrder != "asc" && f.SortOrder != "desc" {
		return shared.NewValidationError("sort order must be 'asc' or 'desc'")
	}

	return nil
}

func (f *TemplateFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	if f.SortBy == "" {
		f.SortBy = "created_at"
	}

	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}

func (f *PublicTemplateFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 1000 {
		return shared.NewValidationError("limit must be between 0 and 1000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	if f.MinRating != nil && (*f.MinRating < 0 || *f.MinRating > 5) {
		return shared.NewValidationError("rating must be between 0 and 5")
	}

	return nil
}

func (f *PublicTemplateFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	if f.SortBy == "" {
		f.SortBy = "downloads"
	}

	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}

// Helper methods for installations
func (i *TemplateInstallation) IsActive() bool {
	return i.Status == InstallationActive
}

func (i *TemplateInstallation) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return shared.Now().After(*i.ExpiresAt)
}

func (i *TemplateInstallation) Uninstall() {
	i.Status = InstallationUninstalled
	i.UpdatedAt = shared.Now()
}

func (i *TemplateInstallation) UpdateUsage() {
	now := shared.Now()
	i.LastUsedAt = &now
	i.UpdatedAt = now
}
