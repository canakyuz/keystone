package handlers

import (
	"github.com/gofiber/fiber/v2"
	"nexspaces-api/internal/core/domain/subscription"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
	uc "nexspaces-api/internal/core/usecases/tenant"
	"nexspaces-api/internal/shared/errors"
	"nexspaces-api/internal/shared/validation"
	"time"
)

// TenantHandler handles tenant-related HTTP requests
type TenantHandler struct {
	createTenantUC *uc.TenantUseCase
	validator      *validation.Validator
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(
	createTenantUC *uc.TenantUseCase,
	validator *validation.Validator,
) *TenantHandler {
	return &TenantHandler{
		createTenantUC: createTenantUC,
		validator:      validator,
	}
}

// CreateTenantRequest represents the HTTP request for creating a tenant
type CreateTenantRequest struct {
	Name            string                 `json:"name" validate:"required,min=2,max=100"`
	Slug            string                 `json:"slug" validate:"required,min=3,max=50,slug"`
	CustomDomain    string                 `json:"custom_domain,omitempty" validate:"omitempty,fqdn"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
	OwnerEmail      string                 `json:"owner_email" validate:"required,email"`
	OwnerFirstName  string                 `json:"owner_first_name" validate:"required,min=1,max=100"`
	OwnerLastName   string                 `json:"owner_last_name" validate:"required,min=1,max=100"`
	PlanID          string                 `json:"plan_id" validate:"required"`
	BillingCycle    string                 `json:"billing_cycle" validate:"required,oneof=monthly yearly"`
	PaymentMethodID string                 `json:"payment_method_id" validate:"required"`
}

// CreateTenantResponse represents the HTTP response for tenant creation
type CreateTenantResponse struct {
	Tenant       TenantDTO       `json:"tenant"`
	Owner        UserDTO         `json:"owner"`
	Subscription SubscriptionDTO `json:"subscription"`
	InviteToken  string          `json:"invite_token"`
}

// TenantDTO represents tenant data transfer object
type TenantDTO struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	CustomDomain *string                `json:"custom_domain,omitempty"`
	Status       string                 `json:"status"`
	Settings     map[string]interface{} `json:"settings"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// UserDTO represents user data transfer object
type UserDTO struct {
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	Email         string  `json:"email"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	EmailVerified bool    `json:"email_verified"`
	LastLoginAt   *string `json:"last_login_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// SubscriptionDTO represents subscription data transfer object
type SubscriptionDTO struct {
	ID                 string                 `json:"id"`
	TenantID           string                 `json:"tenant_id"`
	PlanID             string                 `json:"plan_id"`
	Status             string                 `json:"status"`
	BillingCycle       string                 `json:"billing_cycle"`
	PriceAmount        int64                  `json:"price_amount"`
	PriceCurrency      string                 `json:"price_currency"`
	Features           map[string]interface{} `json:"features"`
	UsageLimits        map[string]interface{} `json:"usage_limits"`
	CurrentUsage       map[string]interface{} `json:"current_usage"`
	CurrentPeriodStart *string                `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *string                `json:"current_period_end,omitempty"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

// CreateTenant handles POST /api/tenants
func (h *TenantHandler) CreateTenant(c *fiber.Ctx) error {
	var req CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Convert to use case request
	ucReq := tenant.CreateTenantRequest{
		Name:         req.Name,
		Slug:         req.Slug,
		CustomDomain: req.CustomDomain,
	}

	// Execute use case
	result, err := h.createTenantUC.CreateTenant(c.Context(), ucReq)
	if err != nil {
		return errors.HandleDomainError(err)
	}

	// Convert to response DTOs
	response := CreateTenantResponse{
		Tenant:       convertTenantToDTO(result.Tenant),
		Owner:        convertUserToDTO(result.Owner),
		Subscription: convertSubscriptionToDTO(result.Subscription),
		InviteToken:  result.InviteToken,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    response,
		"message": "Tenant created successfully",
	})
}

// convertTenantToDTO converts domain tenant to DTO
func convertTenantToDTO(t *tenant.Tenant) TenantDTO {
	dto := TenantDTO{
		ID:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug.String(),
		Status:    string(t.Status),
		Settings:  t.Settings,
		CreatedAt: t.CreatedAt.Time().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt.Time().Format("2006-01-02T15:04:05Z07:00"),
	}

	if t.CustomDomain != nil {
		domain := t.CustomDomain.String()
		dto.CustomDomain = &domain
	}

	return dto
}

// convertUserToDTO converts domain user to DTO
func convertUserToDTO(u *user.User) UserDTO {
	dto := UserDTO{
		ID:            u.ID.String(),
		TenantID:      u.TenantID.String(),
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Role:          string(u.Role),
		Status:        string(u.Status),
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if u.LastLoginAt != nil {
		lastLogin := u.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		dto.LastLoginAt = &lastLogin
	}

	return dto
}

// convertSubscriptionToDTO converts domain subscription to DTO
func convertSubscriptionToDTO(s *subscription.Subscription) SubscriptionDTO {
	dto := SubscriptionDTO{
		ID:            s.ID.String(),
		TenantID:      s.TenantID.String(),
		PlanID:        s.PlanID,
		Status:        string(s.Status),
		BillingCycle:  string(s.BillingCycle),
		PriceAmount:   int64(s.Amount),
		PriceCurrency: s.Currency,
		Features:      make(map[string]interface{}), // Features are on the plan, not subscription
		UsageLimits: map[string]interface{}{
			"templates":      s.UsageLimits.Templates,
			"users":          s.UsageLimits.Users,
			"storage_gb":     s.UsageLimits.Storage,
			"api_requests":   s.UsageLimits.APIRequests,
			"custom_domains": s.UsageLimits.CustomDomains,
		},
		CurrentUsage: map[string]interface{}{
			"templates":      s.CurrentUsage.Templates,
			"users":          s.CurrentUsage.Users,
			"storage_gb":     s.CurrentUsage.Storage,
			"api_requests":   s.CurrentUsage.APIRequests,
			"custom_domains": s.CurrentUsage.CustomDomains,
		},
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if s.StartDate != (time.Time{}) {
		start := s.StartDate.Format("2006-01-02T15:04:05Z07:00")
		dto.CurrentPeriodStart = &start
	}

	if s.EndDate != nil {
		end := s.EndDate.Format("2006-01-02T15:04:05Z07:00")
		dto.CurrentPeriodEnd = &end
	}

	return dto
}
