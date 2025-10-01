package tenant

import (
	"time"

	"nexpaces-api/internal/domain/tenant"
)

// CreateTenantRequest represents request to create a tenant
type CreateTenantRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Slug  string `json:"slug" validate:"required,min=2,max=50,slug"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone,omitempty" validate:"omitempty,max=20"`
	Plan  string `json:"plan" validate:"required,oneof=free starter pro enterprise"`
}

// UpdateTenantRequest represents request to update a tenant
type UpdateTenantRequest struct {
	Name  string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Email string `json:"email,omitempty" validate:"omitempty,email"`
	Phone string `json:"phone,omitempty" validate:"omitempty,max=20"`
}

// UpdateTenantStatusRequest represents request to update tenant status
type UpdateTenantStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active suspended inactive trial"`
	Reason string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// UpgradePlanRequest represents request to upgrade subscription plan
type UpgradePlanRequest struct {
	Plan string `json:"plan" validate:"required,oneof=free starter pro enterprise"`
}

// SetCustomDomainRequest represents request to set custom domain
type SetCustomDomainRequest struct {
	Domain string `json:"domain" validate:"required,domain"`
}

// TenantResponse represents tenant response
type TenantResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Status    string    `json:"status"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Subscription
	SubscriptionStart *time.Time `json:"subscription_start,omitempty"`
	SubscriptionEnd   *time.Time `json:"subscription_end,omitempty"`
	TrialEndsAt       *time.Time `json:"trial_ends_at,omitempty"`

	// Custom domain
	CustomDomain         string     `json:"custom_domain,omitempty"`
	CustomDomainVerified bool       `json:"custom_domain_verified"`
	CustomDomainVerifiedAt *time.Time `json:"custom_domain_verified_at,omitempty"`

	// Feature limits
	FeatureLimits FeatureLimitsResponse `json:"feature_limits"`
}

// FeatureLimitsResponse represents feature limits response
type FeatureLimitsResponse struct {
	MaxUsers     int   `json:"max_users"`
	MaxWebsites  int   `json:"max_websites"`
	MaxStorage   int64 `json:"max_storage"`
	CustomDomain bool  `json:"custom_domain"`
	APIAccess    bool  `json:"api_access"`
}

// TenantListResponse represents paginated tenant list response
type TenantListResponse struct {
	Data       []*TenantResponse `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}

// ToResponse converts domain tenant to response DTO
func ToResponse(t *tenant.Tenant) *TenantResponse {
	limits := tenant.SubscriptionPlan(t.Plan).GetFeatureLimits()

	return &TenantResponse{
		ID:                     t.ID,
		Name:                   t.Name,
		Slug:                   t.Slug,
		Email:                  t.Email,
		Phone:                  t.Phone,
		Status:                 string(t.Status),
		Plan:                   string(t.Plan),
		CreatedAt:              t.CreatedAt,
		UpdatedAt:              t.UpdatedAt,
		SubscriptionStart:      t.SubscriptionStart,
		SubscriptionEnd:        t.SubscriptionEnd,
		TrialEndsAt:            t.TrialEndsAt,
		CustomDomain:           t.CustomDomain,
		CustomDomainVerified:   t.CustomDomainVerified,
		CustomDomainVerifiedAt: t.CustomDomainVerifiedAt,
		FeatureLimits: FeatureLimitsResponse{
			MaxUsers:     limits.MaxUsers,
			MaxWebsites:  limits.MaxWebsites,
			MaxStorage:   limits.MaxStorage,
			CustomDomain: limits.CustomDomain,
			APIAccess:    limits.APIAccess,
		},
	}
}

// ToResponseList converts domain tenants to response list
func ToResponseList(tenants []*tenant.Tenant, total int64, page, perPage int) *TenantListResponse {
	data := make([]*TenantResponse, len(tenants))
	for i, t := range tenants {
		data[i] = ToResponse(t)
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &TenantListResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}
