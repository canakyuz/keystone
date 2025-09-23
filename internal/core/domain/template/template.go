package template

import (
	"fmt"
	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/shared"
)

// Enums for Template properties
type Category string
type Status string
type Visibility string
type PricingModel string
type InstallationStatus string

const (
	CategoryWeb     Category = "web"
	CategoryAPI     Category = "api"
	CategoryMobile  Category = "mobile"
	CategoryDesktop Category = "desktop"
	CategoryOther   Category = "other"
)

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

const (
	PricingModelFree PricingModel = "free"
	PricingModelPaid PricingModel = "paid"
)

const (
	InstallationStatusInstalled   InstallationStatus = "installed"
	InstallationStatusUninstalled InstallationStatus = "uninstalled"
	InstallationStatusFailed      InstallationStatus = "failed"
)

// Template is the main domain entity for a template
type Template struct {
	ID            shared.TemplateID
	TenantID      shared.TenantID
	CreatedBy     shared.UserID
	Name          string
	Description   string
	Category      Category
	Tags          []string
	Configuration shared.JSONB
	Variables     shared.JSONB
	Version       shared.Version
	Status        Status
	Visibility    Visibility
	PricingModel  PricingModel
	PriceAmount   *shared.Money
	InstallCount  int64
	Rating        float64
	RatingCount   int64
	PublishedAt   *shared.Timestamp
	ArchivedAt    *shared.Timestamp
	Metadata      shared.JSONB
	CreatedAt     *shared.Timestamp
	UpdatedAt     *shared.Timestamp
}

// Installation represents an installation of a template by a tenant
type Installation struct {
	ID            shared.InstallationID
	TemplateID    shared.TemplateID
	TenantID      shared.TenantID
	InstalledBy   shared.UserID
	Configuration shared.JSONB
	Status        InstallationStatus
	CreatedAt     *shared.Timestamp
	UninstalledAt *shared.Timestamp
	Metadata      shared.JSONB
}

// Helper functions to parse string values into enum types

func ParseCategory(s string) (Category, error) {
	c := Category(s)
	switch c {
	case CategoryWeb, CategoryAPI, CategoryMobile, CategoryDesktop, CategoryOther:
		return c, nil
	default:
		return "", fmt.Errorf("invalid template category: %s", s)
	}
}

func ParseStatus(s string) (Status, error) {
	st := Status(s)
	switch st {
	case StatusDraft, StatusPublished, StatusArchived:
		return st, nil
	default:
		return "", fmt.Errorf("invalid template status: %s", s)
	}
}

func ParseVisibility(s string) (Visibility, error) {
	v := Visibility(s)
	switch v {
	case VisibilityPublic, VisibilityPrivate:
		return v, nil
	default:
		return "", fmt.Errorf("invalid template visibility: %s", s)
	}
}

func ParsePricingModel(s string) (PricingModel, error) {
	p := PricingModel(s)
	switch p {
	case PricingModelFree, PricingModelPaid:
		return p, nil
	default:
		return "", fmt.Errorf("invalid template pricing model: %s", s)
	}
}

func ParseInstallationStatus(s string) (InstallationStatus, error) {
	is := InstallationStatus(s)
	switch is {
	case InstallationStatusInstalled, InstallationStatusUninstalled, InstallationStatusFailed:
		return is, nil
	default:
		return "", fmt.Errorf("invalid installation status: %s", s)
	}
}

func (t *Template) Publish(userID shared.UserID, releaseNotes string) error {
	t.Status = StatusPublished
	t.PublishedAt = shared.NewTimestamp()
	t.UpdatedAt = shared.NewTimestamp()
	// In a real app, you might want to do more here, like validating the template
	// or creating a new version.
	return nil
}

// NewTemplate creates a new Template instance
func NewTemplate(tenantID shared.TenantID, createdBy shared.UserID, name, description string, category Category, tags []string, configuration shared.JSONB, visibility Visibility, pricingModel PricingModel) (*Template, error) {
	now := shared.NewTimestamp()
	version, err := shared.NewVersion("0.1.0")
	if err != nil {
		return nil, err
	}

	template := &Template{
		ID:            shared.TemplateID(uuid.New()),
		TenantID:      tenantID,
		CreatedBy:     createdBy,
		Name:          name,
		Description:   description,
		Category:      category,
		Tags:          tags,
		Configuration: configuration,
		Version:       *version,
		Status:        StatusDraft,
		Visibility:    visibility,
		PricingModel:  pricingModel,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return template, nil
}
