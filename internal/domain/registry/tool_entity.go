package registry

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ToolStatus represents tool status
type ToolStatus string

const (
	ToolStatusActive     ToolStatus = "active"
	ToolStatusInactive   ToolStatus = "inactive"
	ToolStatusDeprecated ToolStatus = "deprecated"
	ToolStatusArchived   ToolStatus = "archived"
)

// ToolType represents tool type
type ToolType string

const (
	ToolTypeGlobal         ToolType = "global"
	ToolTypeModuleSpecific ToolType = "module_specific"
)

// ToolScope represents tool scope
type ToolScope string

const (
	ToolScopeCrossModule ToolScope = "cross_module"
	ToolScopeModuleBound ToolScope = "module_bound"
)

// ToolCategory represents tool category
type ToolCategory string

const (
	ToolCategoryPayment      ToolCategory = "payment"
	ToolCategoryCommunication ToolCategory = "communication"
	ToolCategoryAutomation   ToolCategory = "automation"
	ToolCategoryStorage      ToolCategory = "storage"
	ToolCategoryAnalytics    ToolCategory = "analytics"
	ToolCategoryNotification ToolCategory = "notification"
	ToolCategoryIntegration  ToolCategory = "integration"
	ToolCategorySecurity     ToolCategory = "security"
	ToolCategoryOther        ToolCategory = "other"
)

// IntegrationType represents integration type
type IntegrationType string

const (
	IntegrationNative     IntegrationType = "native"
	IntegrationThirdParty IntegrationType = "third_party"
	IntegrationCustom     IntegrationType = "custom"
	IntegrationNone       IntegrationType = "none"
)

// Tool represents a platform tool in the catalog
type Tool struct {
	ID                       string           `json:"id"`
	Name                     string           `json:"name"`
	Slug                     string           `json:"slug"`
	Code                     string           `json:"code"` // e.g., "PAYMENT", "CHAT"
	DisplayName              string           `json:"display_name"`
	Description              string           `json:"description"`
	Category                 ToolCategory     `json:"category"`
	ToolType                 ToolType         `json:"tool_type"`
	Scope                    ToolScope        `json:"scope"`
	Status                   ToolStatus       `json:"status"`
	IsPublic                 bool             `json:"is_public"`
	IsBeta                   bool             `json:"is_beta"`
	Version                  string           `json:"version"`
	MinPlatformVersion       string           `json:"min_platform_version,omitempty"`
	PricingModel             PricingModel     `json:"pricing_model"`
	BasePrice                float64          `json:"base_price"`
	Currency                 string           `json:"currency"`
	BillingCycle             string           `json:"billing_cycle,omitempty"`
	TransactionFeePercentage float64          `json:"transaction_fee_percentage,omitempty"`
	TransactionFeeFixed      float64          `json:"transaction_fee_fixed,omitempty"`
	Features                 json.RawMessage  `json:"features"`
	Capabilities             json.RawMessage  `json:"capabilities"`
	Icon                     string           `json:"icon,omitempty"`
	CoverImage               string           `json:"cover_image,omitempty"`
	Screenshots              []string         `json:"screenshots,omitempty"`
	DemoURL                  string           `json:"demo_url,omitempty"`
	DocumentationURL         string           `json:"documentation_url,omitempty"`
	RequiresAPIKeys          bool             `json:"requires_api_keys"`
	RequiresWebhook          bool             `json:"requires_webhook"`
	RequiresStorage          bool             `json:"requires_storage"`
	RequiresDatabase         bool             `json:"requires_database"`
	DatabaseTables           []string         `json:"database_tables,omitempty"`
	IntegrationProvider      string           `json:"integration_provider,omitempty"`
	IntegrationType          IntegrationType  `json:"integration_type"`
	APIEndpoints             json.RawMessage  `json:"api_endpoints,omitempty"`
	DefaultLimits            json.RawMessage  `json:"default_limits"`
	RateLimits               json.RawMessage  `json:"rate_limits"`
	ConfigurationSchema      json.RawMessage  `json:"configuration_schema"`
	DefaultConfiguration     json.RawMessage  `json:"default_configuration"`
	Tags                     []string         `json:"tags,omitempty"`
	Metadata                 json.RawMessage  `json:"metadata"`
	InstallCount             int              `json:"install_count"`
	Rating                   float64          `json:"rating"`
	ReviewCount              int              `json:"review_count"`
	CreatedAt                time.Time        `json:"created_at"`
	UpdatedAt                time.Time        `json:"updated_at"`
	DeletedAt                *time.Time       `json:"deleted_at,omitempty"`
	CreatedBy                string           `json:"created_by,omitempty"`
	UpdatedBy                string           `json:"updated_by,omitempty"`
}

// NewTool creates a new tool
func NewTool(name, slug, code, displayName, description string, category ToolCategory) (*Tool, error) {
	now := time.Now()

	tool := &Tool{
		ID:               uuid.New().String(),
		Name:             name,
		Slug:             slug,
		Code:             code,
		DisplayName:      displayName,
		Description:      description,
		Category:         category,
		ToolType:         ToolTypeGlobal,
		Scope:            ToolScopeCrossModule,
		Status:           ToolStatusActive,
		IsPublic:         true,
		IsBeta:           false,
		Version:          "1.0.0",
		PricingModel:     PricingFree,
		BasePrice:        0.00,
		Currency:         "USD",
		IntegrationType:  IntegrationNative,
		RequiresAPIKeys:  false,
		RequiresWebhook:  false,
		RequiresStorage:  false,
		RequiresDatabase: false,
		InstallCount:     0,
		Rating:           0.00,
		ReviewCount:      0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := tool.Validate(); err != nil {
		return nil, err
	}

	return tool, nil
}

// Validate validates tool data
func (t *Tool) Validate() error {
	if t.ID == "" {
		return ErrInvalidToolID
	}
	if t.Name == "" {
		return ErrToolNameRequired
	}
	if len(t.Name) < 2 || len(t.Name) > 100 {
		return ErrInvalidToolName
	}
	if t.Slug == "" {
		return ErrToolSlugRequired
	}
	if len(t.Slug) < 2 || len(t.Slug) > 50 {
		return ErrInvalidToolSlug
	}
	if t.Code == "" {
		return ErrToolCodeRequired
	}
	if len(t.Code) < 2 || len(t.Code) > 20 {
		return ErrInvalidToolCode
	}
	if !t.Status.IsValid() {
		return ErrInvalidToolStatus
	}
	if !t.ToolType.IsValid() {
		return ErrInvalidToolType
	}
	if !t.Scope.IsValid() {
		return ErrInvalidToolScope
	}
	if !t.PricingModel.IsValid() {
		return ErrInvalidPricingModel
	}
	if !t.Category.IsValid() {
		return ErrInvalidToolCategory
	}
	if !t.IntegrationType.IsValid() {
		return ErrInvalidIntegrationType
	}
	if t.Rating < 0 || t.Rating > 5 {
		return ErrInvalidRating
	}
	if t.BasePrice < 0 {
		return ErrInvalidPrice
	}
	if t.TransactionFeePercentage < 0 || t.TransactionFeePercentage > 100 {
		return ErrInvalidTransactionFee
	}

	return nil
}

// Activate activates the tool
func (t *Tool) Activate() {
	t.Status = ToolStatusActive
	t.UpdatedAt = time.Now()
}

// Deactivate deactivates the tool
func (t *Tool) Deactivate() {
	t.Status = ToolStatusInactive
	t.UpdatedAt = time.Now()
}

// IncrementInstallCount increments the install count
func (t *Tool) IncrementInstallCount() {
	t.InstallCount++
	t.UpdatedAt = time.Now()
}

// DecrementInstallCount decrements the install count
func (t *Tool) DecrementInstallCount() {
	if t.InstallCount > 0 {
		t.InstallCount--
	}
	t.UpdatedAt = time.Now()
}

// IsActive checks if tool is active
func (t *Tool) IsActive() bool {
	return t.Status == ToolStatusActive
}

// IsPubliclyAvailable checks if tool is available in marketplace
func (t *Tool) IsPubliclyAvailable() bool {
	return t.IsPublic && t.Status == ToolStatusActive && t.DeletedAt == nil
}

// IsValid checks if tool status is valid
func (s ToolStatus) IsValid() bool {
	switch s {
	case ToolStatusActive, ToolStatusInactive, ToolStatusDeprecated, ToolStatusArchived:
		return true
	default:
		return false
	}
}

// IsValid checks if tool type is valid
func (t ToolType) IsValid() bool {
	switch t {
	case ToolTypeGlobal, ToolTypeModuleSpecific:
		return true
	default:
		return false
	}
}

// IsValid checks if tool scope is valid
func (s ToolScope) IsValid() bool {
	switch s {
	case ToolScopeCrossModule, ToolScopeModuleBound:
		return true
	default:
		return false
	}
}

// IsValid checks if tool category is valid
func (c ToolCategory) IsValid() bool {
	switch c {
	case ToolCategoryPayment, ToolCategoryCommunication, ToolCategoryAutomation,
		ToolCategoryStorage, ToolCategoryAnalytics, ToolCategoryNotification,
		ToolCategoryIntegration, ToolCategorySecurity, ToolCategoryOther:
		return true
	default:
		return false
	}
}

// IsValid checks if integration type is valid
func (i IntegrationType) IsValid() bool {
	switch i {
	case IntegrationNative, IntegrationThirdParty, IntegrationCustom, IntegrationNone:
		return true
	default:
		return false
	}
}
