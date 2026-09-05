package registry

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModuleStatus represents module status
type ModuleStatus string

const (
	ModuleStatusActive     ModuleStatus = "active"
	ModuleStatusInactive   ModuleStatus = "inactive"
	ModuleStatusDeprecated ModuleStatus = "deprecated"
	ModuleStatusArchived   ModuleStatus = "archived"
)

// ModuleType represents module tier
type ModuleType string

const (
	ModuleTypeStandard   ModuleType = "standard"
	ModuleTypePremium    ModuleType = "premium"
	ModuleTypeEnterprise ModuleType = "enterprise"
)

// PricingModel represents how module is priced
type PricingModel string

const (
	PricingFree         PricingModel = "free"
	PricingOneTime      PricingModel = "one_time"
	PricingSubscription PricingModel = "subscription"
	PricingUsageBased   PricingModel = "usage_based"
)

// ModuleCategory represents module category
type ModuleCategory string

const (
	CategoryEducation     ModuleCategory = "education"
	CategoryContent       ModuleCategory = "content"
	CategoryCommerce      ModuleCategory = "commerce"
	CategoryHospitality   ModuleCategory = "hospitality"
	CategoryManagement    ModuleCategory = "management"
	CategoryCommunication ModuleCategory = "communication"
	CategoryAnalytics     ModuleCategory = "analytics"
	CategoryOther         ModuleCategory = "other"
)

// Module represents a platform module in the catalog
type Module struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Slug                string          `json:"slug"`
	Code                string          `json:"code"` // e.g., "LMS", "CMS"
	DisplayName         string          `json:"display_name"`
	Description         string          `json:"description"`
	Category            ModuleCategory  `json:"category"`
	ModuleType          ModuleType      `json:"module_type"`
	Status              ModuleStatus    `json:"status"`
	IsPublic            bool            `json:"is_public"`
	IsBeta              bool            `json:"is_beta"`
	Version             string          `json:"version"`
	MinPlatformVersion  string          `json:"min_platform_version,omitempty"`
	PricingModel        PricingModel    `json:"pricing_model"`
	BasePrice           float64         `json:"base_price"`
	Currency            string          `json:"currency"`
	BillingCycle        string          `json:"billing_cycle,omitempty"` // monthly, yearly, one_time
	Features            json.RawMessage `json:"features"`                // JSONB array
	Capabilities        json.RawMessage `json:"capabilities"`            // JSONB object
	Icon                string          `json:"icon,omitempty"`
	CoverImage          string          `json:"cover_image,omitempty"`
	Screenshots         []string        `json:"screenshots,omitempty"`
	DemoURL             string          `json:"demo_url,omitempty"`
	DocumentationURL    string          `json:"documentation_url,omitempty"`
	RequiresDatabase    bool            `json:"requires_database"`
	RequiresStorage     bool            `json:"requires_storage"`
	RequiresEmail       bool            `json:"requires_email"`
	DatabaseTables      []string        `json:"database_tables,omitempty"`
	DefaultLimits       json.RawMessage `json:"default_limits"` // JSONB object
	InstallationNotes   string          `json:"installation_notes,omitempty"`
	ConfigurationSchema json.RawMessage `json:"configuration_schema"` // JSON Schema
	Tags                []string        `json:"tags,omitempty"`
	Metadata            json.RawMessage `json:"metadata"` // JSONB object
	InstallCount        int             `json:"install_count"`
	Rating              float64         `json:"rating"`
	ReviewCount         int             `json:"review_count"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           *time.Time      `json:"deleted_at,omitempty"`
	CreatedBy           string          `json:"created_by,omitempty"`
	UpdatedBy           string          `json:"updated_by,omitempty"`
}

// NewModule creates a new module
func NewModule(name, slug, code, displayName, description string, category ModuleCategory) (*Module, error) {
	now := time.Now()

	module := &Module{
		ID:               uuid.New().String(),
		Name:             name,
		Slug:             slug,
		Code:             code,
		DisplayName:      displayName,
		Description:      description,
		Category:         category,
		ModuleType:       ModuleTypeStandard,
		Status:           ModuleStatusActive,
		IsPublic:         true,
		IsBeta:           false,
		Version:          "1.0.0",
		PricingModel:     PricingFree,
		BasePrice:        0.00,
		Currency:         "USD",
		RequiresDatabase: true,
		RequiresStorage:  false,
		RequiresEmail:    false,
		InstallCount:     0,
		Rating:           0.00,
		ReviewCount:      0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := module.Validate(); err != nil {
		return nil, err
	}

	return module, nil
}

// Validate validates module data
func (m *Module) Validate() error {
	if m.ID == "" {
		return ErrInvalidModuleID
	}
	if m.Name == "" {
		return ErrModuleNameRequired
	}
	if len(m.Name) < 2 || len(m.Name) > 100 {
		return ErrInvalidModuleName
	}
	if m.Slug == "" {
		return ErrModuleSlugRequired
	}
	if len(m.Slug) < 2 || len(m.Slug) > 50 {
		return ErrInvalidModuleSlug
	}
	if m.Code == "" {
		return ErrModuleCodeRequired
	}
	if len(m.Code) < 2 || len(m.Code) > 20 {
		return ErrInvalidModuleCode
	}
	if !m.Status.IsValid() {
		return ErrInvalidModuleStatus
	}
	if !m.ModuleType.IsValid() {
		return ErrInvalidModuleType
	}
	if !m.PricingModel.IsValid() {
		return ErrInvalidPricingModel
	}
	if !m.Category.IsValid() {
		return ErrInvalidModuleCategory
	}
	if m.Rating < 0 || m.Rating > 5 {
		return ErrInvalidRating
	}
	if m.BasePrice < 0 {
		return ErrInvalidPrice
	}

	return nil
}

// Activate activates the module
func (m *Module) Activate() {
	m.Status = ModuleStatusActive
	m.UpdatedAt = time.Now()
}

// Deactivate deactivates the module
func (m *Module) Deactivate() {
	m.Status = ModuleStatusInactive
	m.UpdatedAt = time.Now()
}

// Deprecate marks module as deprecated
func (m *Module) Deprecate() {
	m.Status = ModuleStatusDeprecated
	m.UpdatedAt = time.Now()
}

// Archive archives the module
func (m *Module) Archive() {
	m.Status = ModuleStatusArchived
	m.UpdatedAt = time.Now()
}

// IncrementInstallCount increments the install count
func (m *Module) IncrementInstallCount() {
	m.InstallCount++
	m.UpdatedAt = time.Now()
}

// DecrementInstallCount decrements the install count
func (m *Module) DecrementInstallCount() {
	if m.InstallCount > 0 {
		m.InstallCount--
	}
	m.UpdatedAt = time.Now()
}

// UpdateRating updates the module rating
func (m *Module) UpdateRating(rating float64) error {
	if rating < 0 || rating > 5 {
		return ErrInvalidRating
	}
	m.Rating = rating
	m.UpdatedAt = time.Now()
	return nil
}

// IsActive checks if module is active
func (m *Module) IsActive() bool {
	return m.Status == ModuleStatusActive
}

// IsPubliclyAvailable checks if module is available in marketplace
func (m *Module) IsPubliclyAvailable() bool {
	return m.IsPublic && m.Status == ModuleStatusActive && m.DeletedAt == nil
}

// IsFree checks if module is free
func (m *Module) IsFree() bool {
	return m.PricingModel == PricingFree
}

// IsPaid checks if module requires payment
func (m *Module) IsPaid() bool {
	return !m.IsFree()
}

// IsValid checks if module status is valid
func (s ModuleStatus) IsValid() bool {
	switch s {
	case ModuleStatusActive, ModuleStatusInactive, ModuleStatusDeprecated, ModuleStatusArchived:
		return true
	default:
		return false
	}
}

// IsValid checks if module type is valid
func (t ModuleType) IsValid() bool {
	switch t {
	case ModuleTypeStandard, ModuleTypePremium, ModuleTypeEnterprise:
		return true
	default:
		return false
	}
}

// IsValid checks if pricing model is valid
func (p PricingModel) IsValid() bool {
	switch p {
	case PricingFree, PricingOneTime, PricingSubscription, PricingUsageBased:
		return true
	default:
		return false
	}
}

// IsValid checks if module category is valid
func (c ModuleCategory) IsValid() bool {
	switch c {
	case CategoryEducation, CategoryContent, CategoryCommerce, CategoryHospitality,
		CategoryManagement, CategoryCommunication, CategoryAnalytics, CategoryOther:
		return true
	default:
		return false
	}
}

// String returns string representation of status
func (s ModuleStatus) String() string {
	return string(s)
}

// String returns string representation of type
func (t ModuleType) String() string {
	return string(t)
}

// String returns string representation of pricing model
func (p PricingModel) String() string {
	return string(p)
}

// String returns string representation of category
func (c ModuleCategory) String() string {
	return string(c)
}
