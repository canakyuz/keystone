package registry

import (
	"time"

	"nexpaces-api/internal/domain/registry"
)

// ModuleListRequest represents module listing request
type ModuleListRequest struct {
	Category     string `query:"category"`
	ModuleType   string `query:"module_type"`
	PricingModel string `query:"pricing_model"`
	Status       string `query:"status"`
	IsPublic     *bool  `query:"is_public"`
	IsBeta       *bool  `query:"is_beta"`
	SortBy       string `query:"sort_by"`    // name, install_count, rating, created_at
	SortOrder    string `query:"sort_order"` // asc, desc
	Page         int    `query:"page"`
	PerPage      int    `query:"per_page"`
}

// ModuleSearchRequest represents module search request
type ModuleSearchRequest struct {
	Query        string `query:"q" validate:"required,min=2"`
	Category     string `query:"category"`
	PricingModel string `query:"pricing_model"`
	SortBy       string `query:"sort_by"`
	SortOrder    string `query:"sort_order"`
	Page         int    `query:"page"`
	PerPage      int    `query:"per_page"`
}

// ModuleResponse represents module response
type ModuleResponse struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	Slug                string                `json:"slug"`
	Code                string                `json:"code"`
	DisplayName         string                `json:"display_name"`
	Description         string                `json:"description"`
	Category            registry.ModuleCategory `json:"category"`
	ModuleType          registry.ModuleType   `json:"module_type"`
	Status              registry.ModuleStatus `json:"status"`
	IsPublic            bool                  `json:"is_public"`
	IsBeta              bool                  `json:"is_beta"`
	Version             string                `json:"version"`
	MinPlatformVersion  string                `json:"min_platform_version"`
	PricingModel        registry.PricingModel `json:"pricing_model"`
	BasePrice           float64               `json:"base_price,omitempty"`
	Currency            string                `json:"currency,omitempty"`
	BillingCycle        string                `json:"billing_cycle,omitempty"`
	Features            interface{}           `json:"features,omitempty"`
	Capabilities        interface{}           `json:"capabilities,omitempty"`
	Icon                string                `json:"icon,omitempty"`
	CoverImage          string                `json:"cover_image,omitempty"`
	Screenshots         []string              `json:"screenshots,omitempty"`
	DemoURL             string                `json:"demo_url,omitempty"`
	DocumentationURL    string                `json:"documentation_url,omitempty"`
	InstallCount        int                   `json:"install_count"`
	Rating              float64               `json:"rating,omitempty"`
	ReviewCount         int                   `json:"review_count"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

// ModuleDetailResponse represents detailed module response
type ModuleDetailResponse struct {
	ModuleResponse
	RequiresDatabase    bool        `json:"requires_database"`
	RequiresStorage     bool        `json:"requires_storage"`
	RequiresEmail       bool        `json:"requires_email"`
	DatabaseTables      []string    `json:"database_tables,omitempty"`
	DefaultLimits       interface{} `json:"default_limits,omitempty"`
	InstallationNotes   string      `json:"installation_notes,omitempty"`
	ConfigurationSchema interface{} `json:"configuration_schema,omitempty"`
	Tags                []string    `json:"tags,omitempty"`
	Metadata            interface{} `json:"metadata,omitempty"`
}

// ModuleListResponse represents paginated module list response
type ModuleListResponse struct {
	Data       []ModuleResponse `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// ToModuleResponse converts domain module to response
func ToModuleResponse(m *registry.Module) ModuleResponse {
	return ModuleResponse{
		ID:                 m.ID,
		Name:               m.Name,
		Slug:               m.Slug,
		Code:               m.Code,
		DisplayName:        m.DisplayName,
		Description:        m.Description,
		Category:           m.Category,
		ModuleType:         m.ModuleType,
		Status:             m.Status,
		IsPublic:           m.IsPublic,
		IsBeta:             m.IsBeta,
		Version:            m.Version,
		MinPlatformVersion: m.MinPlatformVersion,
		PricingModel:       m.PricingModel,
		BasePrice:          m.BasePrice,
		Currency:           m.Currency,
		BillingCycle:       m.BillingCycle,
		Icon:               m.Icon,
		CoverImage:         m.CoverImage,
		InstallCount:       m.InstallCount,
		Rating:             m.Rating,
		ReviewCount:        m.ReviewCount,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

// ToModuleDetailResponse converts domain module to detailed response
func ToModuleDetailResponse(m *registry.Module) ModuleDetailResponse {
	return ModuleDetailResponse{
		ModuleResponse: ToModuleResponse(m),
		RequiresDatabase:    m.RequiresDatabase,
		RequiresStorage:     m.RequiresStorage,
		RequiresEmail:       m.RequiresEmail,
		InstallationNotes:   m.InstallationNotes,
	}
}
