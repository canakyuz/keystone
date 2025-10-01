package website

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the publication status of a website
type Status string

const (
	StatusDraft       Status = "draft"
	StatusPublished   Status = "published"
	StatusArchived    Status = "archived"
	StatusUnpublished Status = "unpublished"
)

// WebsiteType represents the type/category of website
type WebsiteType string

const (
	TypeCMS         WebsiteType = "cms"
	TypeCRM         WebsiteType = "crm"
	TypeEcommerce   WebsiteType = "ecommerce"
	TypeEducation   WebsiteType = "education"
	TypeHospitality WebsiteType = "hospitality"
	TypeERP         WebsiteType = "erp"
	TypeBlog        WebsiteType = "blog"
	TypePortfolio   WebsiteType = "portfolio"
	TypeCustom      WebsiteType = "custom"
)

// Website represents a multi-tenant website entity
type Website struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    uuid.UUID   `json:"tenant_id"` // ⚠️ CRITICAL: Multi-tenant isolation
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Type        WebsiteType `json:"type"`
	Description *string     `json:"description,omitempty"`
	Status      Status      `json:"status"`
	HomepageID  *uuid.UUID  `json:"homepage_id,omitempty"`

	// Content
	Title   string  `json:"title"`
	Logo    *string `json:"logo,omitempty"`
	Favicon *string `json:"favicon,omitempty"`

	// Domain & URL
	CustomDomain         *string `json:"custom_domain,omitempty"`
	CustomDomainVerified bool    `json:"custom_domain_verified"`
	PrimaryURL           string  `json:"primary_url"`

	// Template
	TemplateID      *uuid.UUID `json:"template_id,omitempty"`
	TemplateName    *string    `json:"template_name,omitempty"`
	TemplateVersion *string    `json:"template_version,omitempty"`

	// Publishing
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	UnpublishedAt *time.Time `json:"unpublished_at,omitempty"`
	ArchivedAt    *time.Time `json:"archived_at,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
}

// New creates a new website
func New(tenantID uuid.UUID, name, slug string, websiteType WebsiteType, createdBy *uuid.UUID) (*Website, error) {
	now := time.Now()
	id := uuid.New()

	website := &Website{
		ID:                   id,
		TenantID:             tenantID,
		Name:                 name,
		Slug:                 slug,
		Type:                 websiteType,
		Status:               StatusDraft,
		Title:                name,
		CustomDomainVerified: false,
		PrimaryURL:           slug + ".nexpaces.com",
		CreatedAt:            now,
		UpdatedAt:            now,
		CreatedBy:            createdBy,
	}

	if err := website.Validate(); err != nil {
		return nil, err
	}

	return website, nil
}

// Validate validates website data
func (w *Website) Validate() error {
	if w.Name == "" {
		return ErrNameRequired
	}

	if len(w.Name) < 2 || len(w.Name) > 100 {
		return ErrInvalidName
	}

	if w.Slug == "" {
		return ErrSlugRequired
	}

	if len(w.Slug) < 2 || len(w.Slug) > 50 {
		return ErrInvalidSlug
	}

	if !w.Type.IsValid() {
		return ErrInvalidWebsiteType
	}

	if !w.Status.IsValid() {
		return ErrInvalidStatus
	}

	return nil
}

// IsPublished returns true if the website is published
func (w *Website) IsPublished() bool {
	return w.Status == StatusPublished
}

// IsDraft checks if the website is in draft status
func (w *Website) IsDraft() bool {
	return w.Status == StatusDraft
}

// IsArchived checks if the website is archived
func (w *Website) IsArchived() bool {
	return w.Status == StatusArchived
}

// HasCustomDomain checks if website has a custom domain
func (w *Website) HasCustomDomain() bool {
	return w.CustomDomain != nil && *w.CustomDomain != ""
}

// IsCustomDomainVerified checks if custom domain is verified
func (w *Website) IsCustomDomainVerified() bool {
	return w.HasCustomDomain() && w.CustomDomainVerified
}

// CanPublish returns true if the website can be published
func (w *Website) CanPublish() bool {
	return w.Status == StatusDraft && w.HomepageID != nil
}

// Publish changes the website status to published
func (w *Website) Publish() error {
	if w.IsPublished() {
		return ErrAlreadyPublished
	}

	if w.IsArchived() {
		return ErrCannotPublishArchived
	}

	now := time.Now()
	w.Status = StatusPublished
	w.PublishedAt = &now
	w.UpdatedAt = now

	return nil
}

// Unpublish unpublishes the website
func (w *Website) Unpublish() error {
	if !w.IsPublished() {
		return ErrNotPublished
	}

	now := time.Now()
	w.Status = StatusUnpublished
	w.UnpublishedAt = &now
	w.UpdatedAt = now

	return nil
}

// Archive changes the website status to archived
func (w *Website) Archive() error {
	if w.Status == StatusArchived {
		return ErrAlreadyArchived
	}

	now := time.Now()
	w.Status = StatusArchived
	w.ArchivedAt = &now
	w.UpdatedAt = now

	return nil
}

// Restore restores an archived website to draft
func (w *Website) Restore() error {
	if !w.IsArchived() {
		return ErrNotArchived
	}

	w.Status = StatusDraft
	w.ArchivedAt = nil
	w.UpdatedAt = time.Now()

	return nil
}

// Update updates the website properties
func (w *Website) Update(name, description *string) {
	if name != nil && *name != "" {
		w.Name = *name
	}
	if description != nil {
		w.Description = description
	}
	w.UpdatedAt = time.Now()
}

// UpdateBasicInfo updates basic website information
func (w *Website) UpdateBasicInfo(name, title string, description *string) {
	if name != "" {
		w.Name = name
	}
	if title != "" {
		w.Title = title
	}
	if description != nil {
		w.Description = description
	}
	w.UpdatedAt = time.Now()
}

// SetCustomDomain sets a custom domain for the website
func (w *Website) SetCustomDomain(domain string) error {
	if domain == "" {
		return ErrInvalidCustomDomain
	}

	w.CustomDomain = &domain
	w.CustomDomainVerified = false
	w.UpdatedAt = time.Now()

	return nil
}

// VerifyCustomDomain marks the custom domain as verified
func (w *Website) VerifyCustomDomain() error {
	if !w.HasCustomDomain() {
		return ErrCustomDomainNotSet
	}

	if w.IsCustomDomainVerified() {
		return ErrCustomDomainAlreadyVerified
	}

	w.CustomDomainVerified = true
	w.PrimaryURL = *w.CustomDomain
	w.UpdatedAt = time.Now()

	return nil
}

// RemoveCustomDomain removes the custom domain
func (w *Website) RemoveCustomDomain() {
	w.CustomDomain = nil
	w.CustomDomainVerified = false
	w.PrimaryURL = w.Slug + ".nexpaces.com"
	w.UpdatedAt = time.Now()
}

// SetTemplate associates a template with the website
func (w *Website) SetTemplate(templateID uuid.UUID, templateName, templateVersion string) {
	w.TemplateID = &templateID
	w.TemplateName = &templateName
	w.TemplateVersion = &templateVersion
	w.UpdatedAt = time.Now()
}

// IsValid checks if website status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived, StatusUnpublished:
		return true
	default:
		return false
	}
}

// IsValid checks if website type is valid
func (t WebsiteType) IsValid() bool {
	switch t {
	case TypeCMS, TypeCRM, TypeEcommerce, TypeEducation, TypeHospitality, TypeERP, TypeBlog, TypePortfolio, TypeCustom:
		return true
	default:
		return false
	}
}

// GetDescription returns a description for the website type
func (t WebsiteType) GetDescription() string {
	switch t {
	case TypeCMS:
		return "Content Management System"
	case TypeCRM:
		return "Customer Relationship Management"
	case TypeEcommerce:
		return "E-commerce Platform"
	case TypeEducation:
		return "Educational Platform"
	case TypeHospitality:
		return "Hospitality Management"
	case TypeERP:
		return "Enterprise Resource Planning"
	case TypeBlog:
		return "Blog Platform"
	case TypePortfolio:
		return "Portfolio Website"
	case TypeCustom:
		return "Custom Website"
	default:
		return "Unknown"
	}
}

// ValidateStatus checks if a status string is valid
func ValidateStatus(s string) bool {
	status := Status(s)
	return status.IsValid()
}
