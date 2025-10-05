package registry

import (
	"time"

	"nexpaces-api/internal/domain/registry"
)

// ToolListRequest represents tool listing request
type ToolListRequest struct {
	Category     string `query:"category"`
	ToolType     string `query:"tool_type"`
	Scope        string `query:"scope"`
	PricingModel string `query:"pricing_model"`
	Status       string `query:"status"`
	IsPublic     *bool  `query:"is_public"`
	IsBeta       *bool  `query:"is_beta"`
	SortBy       string `query:"sort_by"`    // name, install_count, rating, created_at
	SortOrder    string `query:"sort_order"` // asc, desc
	Page         int    `query:"page"`
	PerPage      int    `query:"per_page"`
}

// ToolSearchRequest represents tool search request
type ToolSearchRequest struct {
	Query        string `query:"q" validate:"required,min=2"`
	Category     string `query:"category"`
	Scope        string `query:"scope"`
	PricingModel string `query:"pricing_model"`
	SortBy       string `query:"sort_by"`
	SortOrder    string `query:"sort_order"`
	Page         int    `query:"page"`
	PerPage      int    `query:"per_page"`
}

// ToolResponse represents tool response
type ToolResponse struct {
	ID                       string                `json:"id"`
	Name                     string                `json:"name"`
	Slug                     string                `json:"slug"`
	Code                     string                `json:"code"`
	DisplayName              string                `json:"display_name"`
	Description              string                `json:"description"`
	Category                 registry.ToolCategory `json:"category"`
	ToolType                 registry.ToolType     `json:"tool_type"`
	Scope                    registry.ToolScope    `json:"scope"`
	Status                   registry.ToolStatus   `json:"status"`
	IsPublic                 bool                  `json:"is_public"`
	IsBeta                   bool                  `json:"is_beta"`
	Version                  string                `json:"version"`
	MinPlatformVersion       string                `json:"min_platform_version"`
	PricingModel             registry.PricingModel `json:"pricing_model"`
	BasePrice                float64               `json:"base_price,omitempty"`
	Currency                 string                `json:"currency,omitempty"`
	BillingCycle             string                `json:"billing_cycle,omitempty"`
	TransactionFeePercentage float64               `json:"transaction_fee_percentage,omitempty"`
	TransactionFeeFixed      float64               `json:"transaction_fee_fixed,omitempty"`
	Features                 interface{}           `json:"features,omitempty"`
	Capabilities             interface{}           `json:"capabilities,omitempty"`
	Icon                     string                `json:"icon,omitempty"`
	CoverImage               string                `json:"cover_image,omitempty"`
	Screenshots              []string              `json:"screenshots,omitempty"`
	DemoURL                  string                `json:"demo_url,omitempty"`
	DocumentationURL         string                `json:"documentation_url,omitempty"`
	IntegrationType          registry.IntegrationType `json:"integration_type"`
	IntegrationProvider      string                `json:"integration_provider,omitempty"`
	RequiresAPIKeys          bool                  `json:"requires_api_keys"`
	RequiresWebhook          bool                  `json:"requires_webhook"`
	RequiresStorage          bool                  `json:"requires_storage"`
	RequiresDatabase         bool                  `json:"requires_database"`
	InstallCount             int                   `json:"install_count"`
	Rating                   float64               `json:"rating,omitempty"`
	ReviewCount              int                   `json:"review_count"`
	CreatedAt                time.Time             `json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
}

// ToolDetailResponse represents detailed tool response
type ToolDetailResponse struct {
	ToolResponse
	Features             interface{} `json:"features,omitempty"`
	Capabilities         interface{} `json:"capabilities,omitempty"`
	APIEndpoints         interface{} `json:"api_endpoints,omitempty"`
	DefaultLimits        interface{} `json:"default_limits,omitempty"`
	RateLimits           interface{} `json:"rate_limits,omitempty"`
	ConfigurationSchema  interface{} `json:"configuration_schema,omitempty"`
	DefaultConfiguration interface{} `json:"default_configuration,omitempty"`
	DatabaseTables       []string    `json:"database_tables,omitempty"`
	Tags                 []string    `json:"tags,omitempty"`
	Metadata             interface{} `json:"metadata,omitempty"`
}

// ToolListResponse represents paginated tool list response
type ToolListResponse struct {
	Data       []ToolResponse `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// ToToolResponse converts domain tool to response
func ToToolResponse(t *registry.Tool) ToolResponse {
	return ToolResponse{
		ID:                       t.ID,
		Name:                     t.Name,
		Slug:                     t.Slug,
		Code:                     t.Code,
		DisplayName:              t.DisplayName,
		Description:              t.Description,
		Category:                 t.Category,
		ToolType:                 t.ToolType,
		Scope:                    t.Scope,
		Status:                   t.Status,
		IsPublic:                 t.IsPublic,
		IsBeta:                   t.IsBeta,
		Version:                  t.Version,
		MinPlatformVersion:       t.MinPlatformVersion,
		PricingModel:             t.PricingModel,
		BasePrice:                t.BasePrice,
		Currency:                 t.Currency,
		BillingCycle:             t.BillingCycle,
		TransactionFeePercentage: t.TransactionFeePercentage,
		TransactionFeeFixed:      t.TransactionFeeFixed,
		Icon:                     t.Icon,
		CoverImage:               t.CoverImage,
		IntegrationType:          t.IntegrationType,
		IntegrationProvider:      t.IntegrationProvider,
		RequiresAPIKeys:          t.RequiresAPIKeys,
		RequiresWebhook:          t.RequiresWebhook,
		RequiresStorage:          t.RequiresStorage,
		RequiresDatabase:         t.RequiresDatabase,
		InstallCount:             t.InstallCount,
		Rating:                   t.Rating,
		ReviewCount:              t.ReviewCount,
		CreatedAt:                t.CreatedAt,
		UpdatedAt:                t.UpdatedAt,
	}
}

// ToToolDetailResponse converts domain tool to detailed response
func ToToolDetailResponse(t *registry.Tool) ToolDetailResponse {
	return ToolDetailResponse{
		ToolResponse:         ToToolResponse(t),
		Features:             t.Features,
		Capabilities:         t.Capabilities,
		APIEndpoints:         t.APIEndpoints,
		DefaultLimits:        t.DefaultLimits,
		RateLimits:           t.RateLimits,
		ConfigurationSchema:  t.ConfigurationSchema,
		DefaultConfiguration: t.DefaultConfiguration,
		DatabaseTables:       t.DatabaseTables,
		Tags:                 t.Tags,
		Metadata:             t.Metadata,
	}
}
