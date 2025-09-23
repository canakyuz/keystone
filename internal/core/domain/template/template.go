package template

import (
	"fmt"
	"strings"
	"time"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// TemplateID represents a unique template identifier
type TemplateID = shared.ID

// Template represents a template in the marketplace
type Template struct {
	id            TemplateID
	name          string
	slug          shared.Slug
	description   string
	category      Category
	version       string
	authorID      user.UserID
	tenantID      *tenant.TenantID // nil for public templates
	visibility    Visibility
	status        TemplateStatus
	pricing       Pricing
	tags          []string
	screenshot    string
	demoURL       *string
	sourceURL     *string
	documentation string
	metadata      TemplateMetadata
	requirements  Requirements
	configuration Configuration
	statistics    Statistics
	createdAt     shared.Timestamp
	updatedAt     shared.Timestamp
	publishedAt   *shared.Timestamp
	version_int   int64 // For optimistic locking
}

// Category represents template categories
type Category string

const (
	CategoryCMS         Category = "cms"
	CategoryCRM         Category = "crm"
	CategoryEcommerce   Category = "ecommerce"
	CategoryEducation   Category = "education"
	CategoryHospitality Category = "hospitality"
	CategoryERP         Category = "erp"
	CategoryBlog        Category = "blog"
	CategoryPortfolio   Category = "portfolio"
	CategoryLanding     Category = "landing"
	CategoryDashboard   Category = "dashboard"
	CategoryOther       Category = "other"
)

// IsValid checks if category is valid
func (c Category) IsValid() bool {
	switch c {
	case CategoryCMS, CategoryCRM, CategoryEcommerce, CategoryEducation,
		CategoryHospitality, CategoryERP, CategoryBlog, CategoryPortfolio,
		CategoryLanding, CategoryDashboard, CategoryOther:
		return true
	default:
		return false
	}
}

// Visibility represents template visibility
type Visibility string

const (
	VisibilityPublic  Visibility = "public"  // Available to all tenants
	VisibilityPrivate Visibility = "private" // Only for owner tenant
	VisibilityShared  Visibility = "shared"  // Shared with specific tenants
)

// IsValid checks if visibility is valid
func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate, VisibilityShared:
		return true
	default:
		return false
	}
}

// TemplateStatus represents template status
type TemplateStatus string

const (
	StatusDraft     TemplateStatus = "draft"
	StatusReview    TemplateStatus = "review"
	StatusApproved  TemplateStatus = "approved"
	StatusPublished TemplateStatus = "published"
	StatusRejected  TemplateStatus = "rejected"
	StatusArchived  TemplateStatus = "archived"
)

// IsValid checks if template status is valid
func (s TemplateStatus) IsValid() bool {
	switch s {
	case StatusDraft, StatusReview, StatusApproved, StatusPublished, StatusRejected, StatusArchived:
		return true
	default:
		return false
	}
}

// Pricing represents template pricing model
type Pricing struct {
	Model    PricingModel  `json:"model"`
	Price    *shared.Money `json:"price,omitempty"`    // For paid templates
	Currency string        `json:"currency,omitempty"` // Currency code
	Features []string      `json:"features,omitempty"` // What's included
}

// PricingModel represents pricing models
type PricingModel string

const (
	PricingFree     PricingModel = "free"
	PricingOneTime  PricingModel = "one_time"
	PricingMonthly  PricingModel = "monthly"
	PricingYearly   PricingModel = "yearly"
	PricingFreemium PricingModel = "freemium"
)

// IsValid checks if pricing model is valid
func (p PricingModel) IsValid() bool {
	switch p {
	case PricingFree, PricingOneTime, PricingMonthly, PricingYearly, PricingFreemium:
		return true
	default:
		return false
	}
}

// TemplateMetadata contains template metadata
type TemplateMetadata struct {
	Framework     string            `json:"framework"`     // React, Vue, Angular, etc.
	Language      string            `json:"language"`      // JavaScript, TypeScript, etc.
	Database      string            `json:"database"`      // PostgreSQL, MySQL, etc.
	Dependencies  []string          `json:"dependencies"`  // Required dependencies
	Features      []string          `json:"features"`      // Template features
	License       string            `json:"license"`       // MIT, GPL, etc.
	Size          int64             `json:"size"`          // Template size in bytes
	Compatibility []string          `json:"compatibility"` // Compatible versions
	CustomFields  map[string]string `json:"custom_fields"` // Additional metadata
}

// Requirements represents template requirements
type Requirements struct {
	MinMemory     int      `json:"min_memory"`     // MB
	MinStorage    int      `json:"min_storage"`    // MB
	MinBandwidth  int      `json:"min_bandwidth"`  // MB/month
	RequiredPorts []int    `json:"required_ports"` // Required ports
	Environment   []string `json:"environment"`    // Required environment variables
	Services      []string `json:"services"`       // Required services (Redis, etc.)
}

// Configuration represents template configuration options
type Configuration struct {
	Variables    []ConfigVariable `json:"variables"`     // Configuration variables
	Hooks        []Hook           `json:"hooks"`         // Installation hooks
	Customizable []string         `json:"customizable"`  // Customizable components
	DefaultTheme string           `json:"default_theme"` // Default theme
}

// ConfigVariable represents a configuration variable
type ConfigVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, boolean, array
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value"`
	Required     bool        `json:"required"`
	Options      []string    `json:"options,omitempty"`    // For select type
	Validation   string      `json:"validation,omitempty"` // Validation regex
}

// Hook represents installation/setup hooks
type Hook struct {
	Name        string            `json:"name"`
	Stage       string            `json:"stage"` // pre_install, post_install, etc.
	Command     string            `json:"command"`
	Environment map[string]string `json:"environment"`
	Timeout     int               `json:"timeout"` // seconds
}

// Statistics represents template usage statistics
type Statistics struct {
	Downloads   int64   `json:"downloads"`
	Installs    int64   `json:"installs"`
	Rating      float64 `json:"rating"` // 0-5 stars
	RatingCount int     `json:"rating_count"`
	Views       int64   `json:"views"`
	Forks       int     `json:"forks"`
	Stars       int     `json:"stars"`
}

// TemplateParams contains parameters for creating a new template
type TemplateParams struct {
	Name          string
	Slug          string
	Description   string
	Category      Category
	Version       string
	AuthorID      user.UserID
	TenantID      *tenant.TenantID
	Visibility    Visibility
	Pricing       Pricing
	Tags          []string
	Screenshot    string
	DemoURL       *string
	SourceURL     *string
	Documentation string
	Metadata      TemplateMetadata
	Requirements  Requirements
	Configuration Configuration
}

// NewTemplate creates a new template entity
func NewTemplate(params TemplateParams) (*Template, error) {
	// Validate required fields
	if strings.TrimSpace(params.Name) == "" {
		return nil, shared.NewValidationError("template name is required")
	}

	if len(params.Name) > 255 {
		return nil, shared.NewValidationError("template name must be 255 characters or less")
	}

	// Create slug
	slug, err := shared.NewSlug(params.Slug)
	if err != nil {
		return nil, shared.WrapDomainError(err, shared.ValidationError, "invalid template slug")
	}

	// Validate category
	if !params.Category.IsValid() {
		return nil, shared.NewValidationError("invalid template category")
	}

	// Validate visibility
	if !params.Visibility.IsValid() {
		return nil, shared.NewValidationError("invalid template visibility")
	}

	// Validate version
	if strings.TrimSpace(params.Version) == "" {
		return nil, shared.NewValidationError("template version is required")
	}

	// Validate author ID
	if params.AuthorID.IsZero() {
		return nil, shared.NewValidationError("author ID is required")
	}

	// Validate pricing
	if err := validatePricing(params.Pricing); err != nil {
		return nil, shared.WrapDomainError(err, shared.ValidationError, "invalid pricing")
	}

	// Validate public template requirements
	if params.Visibility == VisibilityPublic && params.TenantID != nil {
		return nil, shared.NewValidationError("public templates cannot have tenant ownership")
	}

	// Validate private template requirements
	if params.Visibility == VisibilityPrivate && params.TenantID == nil {
		return nil, shared.NewValidationError("private templates must have tenant ownership")
	}

	now := shared.Now()

	return &Template{
		id:            shared.NewID(),
		name:          strings.TrimSpace(params.Name),
		slug:          slug,
		description:   params.Description,
		category:      params.Category,
		version:       params.Version,
		authorID:      params.AuthorID,
		tenantID:      params.TenantID,
		visibility:    params.Visibility,
		status:        StatusDraft,
		pricing:       params.Pricing,
		tags:          params.Tags,
		screenshot:    params.Screenshot,
		demoURL:       params.DemoURL,
		sourceURL:     params.SourceURL,
		documentation: params.Documentation,
		metadata:      params.Metadata,
		requirements:  params.Requirements,
		configuration: params.Configuration,
		statistics:    Statistics{}, // Initialize empty statistics
		createdAt:     now,
		updatedAt:     now,
		publishedAt:   nil,
		version_int:   1,
	}, nil
}

// ReConstituteTemplate recreates a template from stored data
func ReConstituteTemplate(
	id TemplateID,
	name string,
	slug shared.Slug,
	description string,
	category Category,
	version string,
	authorID user.UserID,
	tenantID *tenant.TenantID,
	visibility Visibility,
	status TemplateStatus,
	pricing Pricing,
	tags []string,
	screenshot string,
	demoURL *string,
	sourceURL *string,
	documentation string,
	metadata TemplateMetadata,
	requirements Requirements,
	configuration Configuration,
	statistics Statistics,
	createdAt shared.Timestamp,
	updatedAt shared.Timestamp,
	publishedAt *shared.Timestamp,
	version_int int64,
) *Template {
	return &Template{
		id:            id,
		name:          name,
		slug:          slug,
		description:   description,
		category:      category,
		version:       version,
		authorID:      authorID,
		tenantID:      tenantID,
		visibility:    visibility,
		status:        status,
		pricing:       pricing,
		tags:          tags,
		screenshot:    screenshot,
		demoURL:       demoURL,
		sourceURL:     sourceURL,
		documentation: documentation,
		metadata:      metadata,
		requirements:  requirements,
		configuration: configuration,
		statistics:    statistics,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		publishedAt:   publishedAt,
		version_int:   version_int,
	}
}

// Getters
func (t *Template) ID() TemplateID {
	return t.id
}

func (t *Template) Name() string {
	return t.name
}

func (t *Template) Slug() shared.Slug {
	return t.slug
}

func (t *Template) Description() string {
	return t.description
}

func (t *Template) Category() Category {
	return t.category
}

func (t *Template) Version() string {
	return t.version
}

func (t *Template) AuthorID() user.UserID {
	return t.authorID
}

func (t *Template) TenantID() *tenant.TenantID {
	return t.tenantID
}

func (t *Template) Visibility() Visibility {
	return t.visibility
}

func (t *Template) Status() TemplateStatus {
	return t.status
}

func (t *Template) Pricing() Pricing {
	return t.pricing
}

func (t *Template) Tags() []string {
	return t.tags
}

func (t *Template) Screenshot() string {
	return t.screenshot
}

func (t *Template) DemoURL() *string {
	return t.demoURL
}

func (t *Template) SourceURL() *string {
	return t.sourceURL
}

func (t *Template) Documentation() string {
	return t.documentation
}

func (t *Template) Metadata() TemplateMetadata {
	return t.metadata
}

func (t *Template) Requirements() Requirements {
	return t.requirements
}

func (t *Template) Configuration() Configuration {
	return t.configuration
}

func (t *Template) Statistics() Statistics {
	return t.statistics
}

func (t *Template) CreatedAt() shared.Timestamp {
	return t.createdAt
}

func (t *Template) UpdatedAt() shared.Timestamp {
	return t.updatedAt
}

func (t *Template) PublishedAt() *shared.Timestamp {
	return t.publishedAt
}

func (t *Template) VersionInt() int64 {
	return t.version_int
}

// Business Methods

// UpdateBasicInfo updates basic template information
func (t *Template) UpdateBasicInfo(name, description string) error {
	if strings.TrimSpace(name) == "" {
		return shared.NewValidationError("template name cannot be empty")
	}

	if len(name) > 255 {
		return shared.NewValidationError("template name must be 255 characters or less")
	}

	t.name = strings.TrimSpace(name)
	t.description = description
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// UpdateVersion updates template version
func (t *Template) UpdateVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return shared.NewValidationError("version cannot be empty")
	}

	t.version = version
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// UpdatePricing updates template pricing
func (t *Template) UpdatePricing(pricing Pricing) error {
	if err := validatePricing(pricing); err != nil {
		return shared.WrapDomainError(err, shared.ValidationError, "invalid pricing")
	}

	t.pricing = pricing
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// UpdateTags updates template tags
func (t *Template) UpdateTags(tags []string) error {
	// Validate and clean tags
	cleanTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag != "" && len(tag) <= 50 {
			cleanTags = append(cleanTags, tag)
		}
	}

	if len(cleanTags) > 20 {
		return shared.NewValidationError("too many tags (maximum 20)")
	}

	t.tags = cleanTags
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// SubmitForReview submits template for review
func (t *Template) SubmitForReview() error {
	if t.status != StatusDraft {
		return shared.NewBusinessRuleError("only draft templates can be submitted for review")
	}

	// Validate required fields for review
	if err := t.validateForReview(); err != nil {
		return err
	}

	t.status = StatusReview
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// Approve approves template after review
func (t *Template) Approve() error {
	if t.status != StatusReview {
		return shared.NewBusinessRuleError("only templates under review can be approved")
	}

	t.status = StatusApproved
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// Reject rejects template after review
func (t *Template) Reject(reason string) error {
	if t.status != StatusReview {
		return shared.NewBusinessRuleError("only templates under review can be rejected")
	}

	t.status = StatusRejected
	t.updatedAt = shared.Now()
	t.version_int++

	// Store rejection reason in metadata
	if t.metadata.CustomFields == nil {
		t.metadata.CustomFields = make(map[string]string)
	}
	t.metadata.CustomFields["rejection_reason"] = reason

	return nil
}

// Publish publishes approved template
func (t *Template) Publish() error {
	if t.status != StatusApproved {
		return shared.NewBusinessRuleError("only approved templates can be published")
	}

	now := shared.Now()
	t.status = StatusPublished
	t.publishedAt = &now
	t.updatedAt = now
	t.version_int++

	return nil
}

// Archive archives template
func (t *Template) Archive() error {
	if t.status == StatusArchived {
		return nil // Already archived
	}

	t.status = StatusArchived
	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// IncrementDownloads increments download count
func (t *Template) IncrementDownloads() {
	t.statistics.Downloads++
	t.updatedAt = shared.Now()
	t.version_int++
}

// IncrementInstalls increments install count
func (t *Template) IncrementInstalls() {
	t.statistics.Installs++
	t.updatedAt = shared.Now()
	t.version_int++
}

// IncrementViews increments view count
func (t *Template) IncrementViews() {
	t.statistics.Views++
	t.updatedAt = shared.Now()
	t.version_int++
}

// UpdateRating updates template rating
func (t *Template) UpdateRating(rating float64) error {
	if rating < 0 || rating > 5 {
		return shared.NewValidationError("rating must be between 0 and 5")
	}

	// Simple rating calculation (can be enhanced with weighted average)
	totalRating := t.statistics.Rating * float64(t.statistics.RatingCount)
	totalRating += rating
	t.statistics.RatingCount++
	t.statistics.Rating = totalRating / float64(t.statistics.RatingCount)

	t.updatedAt = shared.Now()
	t.version_int++

	return nil
}

// Business Logic Queries

// IsPublic checks if template is public
func (t *Template) IsPublic() bool {
	return t.visibility == VisibilityPublic
}

// IsPrivate checks if template is private
func (t *Template) IsPrivate() bool {
	return t.visibility == VisibilityPrivate
}

// IsPublished checks if template is published
func (t *Template) IsPublished() bool {
	return t.status == StatusPublished
}

// IsFree checks if template is free
func (t *Template) IsFree() bool {
	return t.pricing.Model == PricingFree
}

// CanBeAccessedByTenant checks if template can be accessed by tenant
func (t *Template) CanBeAccessedByTenant(tenantID tenant.TenantID) bool {
	// Public templates are accessible by all
	if t.IsPublic() {
		return true
	}

	// Private templates only accessible by owner tenant
	if t.IsPrivate() && t.tenantID != nil {
		return *t.tenantID == tenantID
	}

	// Shared templates (implement sharing logic as needed)
	if t.visibility == VisibilityShared {
		// This would check a sharing table or metadata
		return false // Placeholder
	}

	return false
}

// GetInstallationPrice returns installation price for tenant
func (t *Template) GetInstallationPrice() *shared.Money {
	if t.IsFree() {
		return nil
	}

	return t.pricing.Price
}

// Utility functions

// validatePricing validates pricing information
func validatePricing(pricing Pricing) error {
	if !pricing.Model.IsValid() {
		return fmt.Errorf("invalid pricing model")
	}

	// For paid models, price is required
	if pricing.Model != PricingFree && pricing.Price == nil {
		return fmt.Errorf("price is required for paid templates")
	}

	// For free models, price should be nil
	if pricing.Model == PricingFree && pricing.Price != nil {
		return fmt.Errorf("price should not be set for free templates")
	}

	return nil
}

// validateForReview validates template is ready for review
func (t *Template) validateForReview() error {
	if t.name == "" {
		return shared.NewValidationError("name is required")
	}

	if t.description == "" {
		return shared.NewValidationError("description is required")
	}

	if t.screenshot == "" {
		return shared.NewValidationError("screenshot is required")
	}

	if len(t.tags) == 0 {
		return shared.NewValidationError("at least one tag is required")
	}

	return nil
}

// Events

// TemplateCreatedEvent represents a template creation event
type TemplateCreatedEvent struct {
	TemplateID TemplateID
	Name       string
	AuthorID   user.UserID
	TenantID   *tenant.TenantID
	Category   string
	CreatedAt  time.Time
}

// TemplatePublishedEvent represents a template publication event
type TemplatePublishedEvent struct {
	TemplateID  TemplateID
	Name        string
	AuthorID    user.UserID
	Category    string
	PublishedAt time.Time
}

// TemplateInstalledEvent represents a template installation event
type TemplateInstalledEvent struct {
	TemplateID  TemplateID
	TenantID    tenant.TenantID
	UserID      user.UserID
	Price       *shared.Money
	InstalledAt time.Time
}

// Generate events
func (t *Template) GenerateCreatedEvent() TemplateCreatedEvent {
	return TemplateCreatedEvent{
		TemplateID: t.id,
		Name:       t.name,
		AuthorID:   t.authorID,
		TenantID:   t.tenantID,
		Category:   string(t.category),
		CreatedAt:  t.createdAt.Time(),
	}
}

func (t *Template) GeneratePublishedEvent() TemplatePublishedEvent {
	return TemplatePublishedEvent{
		TemplateID:  t.id,
		Name:        t.name,
		AuthorID:    t.authorID,
		Category:    string(t.category),
		PublishedAt: t.publishedAt.Time(),
	}
}

func (t *Template) GenerateInstalledEvent(tenantID tenant.TenantID, userID user.UserID) TemplateInstalledEvent {
	return TemplateInstalledEvent{
		TemplateID:  t.id,
		TenantID:    tenantID,
		UserID:      userID,
		Price:       t.pricing.Price,
		InstalledAt: shared.Now().Time(),
	}
}
