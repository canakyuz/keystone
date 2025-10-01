package website

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the publication status of a website
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

// Website represents a multi-tenant website entity
type Website struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description,omitempty"`
	Status      Status     `json:"status"`
	HomepageID  *uuid.UUID `json:"homepage_id,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// IsPublished returns true if the website is published
func (w *Website) IsPublished() bool {
	return w.Status == StatusPublished
}

// CanPublish returns true if the website can be published
func (w *Website) CanPublish() bool {
	return w.Status == StatusDraft && w.HomepageID != nil
}

// Publish changes the website status to published
func (w *Website) Publish() error {
	if !w.CanPublish() {
		return ErrCannotPublish
	}
	w.Status = StatusPublished
	w.UpdatedAt = time.Now()
	return nil
}

// Archive changes the website status to archived
func (w *Website) Archive() error {
	if w.Status == StatusArchived {
		return ErrAlreadyArchived
	}
	w.Status = StatusArchived
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

// ValidateStatus checks if a status string is valid
func ValidateStatus(s string) bool {
	status := Status(s)
	return status == StatusDraft || status == StatusPublished || status == StatusArchived
}
