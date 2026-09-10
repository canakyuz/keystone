package project

import (
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusInProgress ProjectStatus = "in-progress"
	ProjectStatusPlanned    ProjectStatus = "planned"
	ProjectStatusCancelled  ProjectStatus = "cancelled"
)

// ProjectCategory represents project categories
type ProjectCategory string

const (
	CategoryWebDesign  ProjectCategory = "web-design"
	CategoryMobileApp  ProjectCategory = "mobile-app"
	CategoryAutomation ProjectCategory = "automation"
	CategoryECommerce  ProjectCategory = "e-commerce"
	CategoryCMS        ProjectCategory = "cms"
	CategoryAPI        ProjectCategory = "api"
	CategoryOther      ProjectCategory = "other"
)

// Project represents a portfolio project
type Project struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"` // CRITICAL: Multi-tenant isolation
	Title        string                 `json:"title"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	Content      string                 `json:"content"` // Rich text content
	Category     ProjectCategory        `json:"category"`
	Status       ProjectStatus          `json:"status"`
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
	CreatedBy    string                 `json:"created_by,omitempty"`
	UpdatedBy    string                 `json:"updated_by,omitempty"`
	DeletedAt    *time.Time             `json:"deleted_at,omitempty"`
}

// New creates a new project
func New(tenantID, title, slug, description string, category ProjectCategory, status ProjectStatus) (*Project, error) {
	now := time.Now()
	project := &Project{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Title:       title,
		Slug:        slug,
		Description: description,
		Category:    category,
		Status:      status,
		Featured:    false,
		SortOrder:   0,
		ViewCount:   0,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := project.Validate(); err != nil {
		return nil, err
	}

	return project, nil
}

// Validate validates project data
func (p *Project) Validate() error {
	if p.ID == "" {
		return ErrInvalidProjectID
	}

	if p.TenantID == "" {
		return ErrTenantIDRequired
	}

	if p.Title == "" {
		return ErrTitleRequired
	}

	if len(p.Title) < 3 || len(p.Title) > 200 {
		return ErrInvalidTitle
	}

	if p.Slug == "" {
		return ErrSlugRequired
	}

	if len(p.Slug) < 3 || len(p.Slug) > 100 {
		return ErrInvalidSlug
	}

	if !p.Category.IsValid() {
		return ErrInvalidCategory
	}

	if !p.Status.IsValid() {
		return ErrInvalidStatus
	}

	return nil
}

// MarkAsCompleted marks project as completed
func (p *Project) MarkAsCompleted() error {
	if p.Status == ProjectStatusCompleted {
		return ErrProjectAlreadyCompleted
	}

	p.Status = ProjectStatusCompleted
	now := time.Now()
	p.EndDate = &now
	p.UpdatedAt = now

	return nil
}

// MarkAsInProgress marks project as in progress
func (p *Project) MarkAsInProgress() error {
	if p.Status == ProjectStatusInProgress {
		return ErrProjectAlreadyInProgress
	}

	p.Status = ProjectStatusInProgress
	now := time.Now()
	if p.StartDate == nil {
		p.StartDate = &now
	}
	p.UpdatedAt = now

	return nil
}

// Cancel cancels the project
func (p *Project) Cancel(reason string) error {
	if p.Status == ProjectStatusCancelled {
		return ErrProjectAlreadyCancelled
	}

	p.Status = ProjectStatusCancelled
	p.UpdatedAt = time.Now()

	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}
	p.Metadata["cancellation_reason"] = reason
	p.Metadata["cancelled_at"] = time.Now()

	return nil
}

// SetFeatured sets or unsets project as featured
func (p *Project) SetFeatured(featured bool) {
	p.Featured = featured
	p.UpdatedAt = time.Now()
}

// IncrementViewCount increments the view count
func (p *Project) IncrementViewCount() {
	p.ViewCount++
	p.UpdatedAt = time.Now()
}

// UpdateContent updates project content
func (p *Project) UpdateContent(title, description, content string) error {
	if title != "" {
		if len(title) < 3 || len(title) > 200 {
			return ErrInvalidTitle
		}
		p.Title = title
	}

	if description != "" {
		p.Description = description
	}

	if content != "" {
		p.Content = content
	}

	p.UpdatedAt = time.Now()
	return nil
}

// AddImage adds an image to the project
func (p *Project) AddImage(imageURL string) {
	if p.Images == nil {
		p.Images = []string{}
	}
	p.Images = append(p.Images, imageURL)
	p.UpdatedAt = time.Now()
}

// RemoveImage removes an image from the project
func (p *Project) RemoveImage(imageURL string) {
	if p.Images == nil {
		return
	}

	for i, img := range p.Images {
		if img == imageURL {
			p.Images = append(p.Images[:i], p.Images[i+1:]...)
			break
		}
	}
	p.UpdatedAt = time.Now()
}

// AddTechnology adds a technology to the project
func (p *Project) AddTechnology(tech string) {
	if p.Technologies == nil {
		p.Technologies = []string{}
	}

	// Check if already exists
	for _, t := range p.Technologies {
		if t == tech {
			return
		}
	}

	p.Technologies = append(p.Technologies, tech)
	p.UpdatedAt = time.Now()
}

// RemoveTechnology removes a technology from the project
func (p *Project) RemoveTechnology(tech string) {
	if p.Technologies == nil {
		return
	}

	for i, t := range p.Technologies {
		if t == tech {
			p.Technologies = append(p.Technologies[:i], p.Technologies[i+1:]...)
			break
		}
	}
	p.UpdatedAt = time.Now()
}

// IsCompleted checks if project is completed
func (p *Project) IsCompleted() bool {
	return p.Status == ProjectStatusCompleted
}

// IsInProgress checks if project is in progress
func (p *Project) IsInProgress() bool {
	return p.Status == ProjectStatusInProgress
}

// IsPlanned checks if project is planned
func (p *Project) IsPlanned() bool {
	return p.Status == ProjectStatusPlanned
}

// IsCancelled checks if project is cancelled
func (p *Project) IsCancelled() bool {
	return p.Status == ProjectStatusCancelled
}

// IsValid checks if project status is valid
func (s ProjectStatus) IsValid() bool {
	switch s {
	case ProjectStatusCompleted, ProjectStatusInProgress, ProjectStatusPlanned, ProjectStatusCancelled:
		return true
	default:
		return false
	}
}

// IsValid checks if project category is valid
func (c ProjectCategory) IsValid() bool {
	switch c {
	case CategoryWebDesign, CategoryMobileApp, CategoryAutomation, CategoryECommerce, CategoryCMS, CategoryAPI, CategoryOther:
		return true
	default:
		return false
	}
}

// String returns string representation of status
func (s ProjectStatus) String() string {
	return string(s)
}

// String returns string representation of category
func (c ProjectCategory) String() string {
	return string(c)
}
