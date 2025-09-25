package handlers

import (
	_ "strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/template"
	templateUC "nexspaces-api/internal/core/usecases/template"
	"nexspaces-api/internal/shared/errors"
	"nexspaces-api/internal/shared/validation"
)

// TemplateHandler handles template-related HTTP requests
type TemplateHandler struct {
	createTemplateUC  *templateUC.CreateTemplateUseCase
	publishTemplateUC *templateUC.PublishTemplateUseCase
	installTemplateUC *templateUC.InstallTemplateUseCase
	searchTemplatesUC *templateUC.SearchTemplatesUseCase
	validator         *validation.Validator
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(
	createTemplateUC *templateUC.CreateTemplateUseCase,
	publishTemplateUC *templateUC.PublishTemplateUseCase,
	installTemplateUC *templateUC.InstallTemplateUseCase,
	searchTemplatesUC *templateUC.SearchTemplatesUseCase,
	validator *validation.Validator,
) *TemplateHandler {
	return &TemplateHandler{
		createTemplateUC:  createTemplateUC,
		publishTemplateUC: publishTemplateUC,
		installTemplateUC: installTemplateUC,
		searchTemplatesUC: searchTemplatesUC,
		validator:         validator,
	}
}

// CreateTemplateRequest represents the HTTP request for creating a template
type CreateTemplateRequest struct {
	Name          string                 `json:"name" validate:"required,min=3,max=100"`
	Description   string                 `json:"description" validate:"required,min=10,max=1000"`
	Category      string                 `json:"category" validate:"required,oneof=cms crm ecommerce education hospitality erp portfolio blog landing dashboard"`
	Tags          []string               `json:"tags,omitempty" validate:"max=10"`
	Configuration map[string]interface{} `json:"configuration" validate:"required"`
	Visibility    string                 `json:"visibility" validate:"required,oneof=private public tenant"`
	PricingModel  string                 `json:"pricing_model" validate:"required,oneof=free one_time subscription"`
}

// PublishTemplateRequest represents the HTTP request for publishing a template
type PublishTemplateRequest struct {
	ReleaseNotes string `json:"release_notes,omitempty" validate:"max=500"`
}

// InstallTemplateRequest represents the HTTP request for installing a template
type InstallTemplateRequest struct {
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// TemplateDTO represents template data transfer object
type TemplateDTO struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	CreatedBy     string                 `json:"created_by"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	Tags          []string               `json:"tags"`
	Configuration map[string]interface{} `json:"configuration"`
	Variables     map[string]interface{} `json:"variables"`
	Version       string                 `json:"version"`
	Status        string                 `json:"status"`
	Visibility    string                 `json:"visibility"`
	PricingModel  string                 `json:"pricing_model"`
	PriceAmount   *int64                 `json:"price_amount,omitempty"`
	PriceCurrency *string                `json:"price_currency,omitempty"`
	InstallCount  int64                  `json:"install_count"`
	Rating        float64                `json:"rating"`
	RatingCount   int64                  `json:"rating_count"`
	PublishedAt   *string                `json:"published_at,omitempty"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// InstallationDTO represents template installation data transfer object
type InstallationDTO struct {
	ID            string                 `json:"id"`
	TemplateID    string                 `json:"template_id"`
	TenantID      string                 `json:"tenant_id"`
	InstalledBy   string                 `json:"installed_by"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        string                 `json:"status"`
	InstalledAt   string                 `json:"installed_at"`
}

// CreateTemplate handles POST /api/tenants/{tenantId}/templates
func (h *TemplateHandler) CreateTemplate(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Extract current user from context
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	var req CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Parse category
	category, err := template.ParseCategory(req.Category)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid category", err)
	}

	// Parse visibility
	visibility, err := template.ParseVisibility(req.Visibility)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid visibility", err)
	}

	// Parse pricing model
	pricingModel, err := template.ParsePricingModel(req.PricingModel)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid pricing model", err)
	}

	// Convert to use case request
	ucReq := templateUC.CreateTemplateRequest{
		TenantID:      tenantID,
		CreatedBy:     shared.UserID(currentUserID),
		Name:          req.Name,
		Description:   req.Description,
		Category:      category,
		Tags:          req.Tags,
		Configuration: req.Configuration,
		Visibility:    visibility,
		PricingModel:  pricingModel,
	}

	// Execute use case
	result, err := h.createTemplateUC.Execute(c.Context(), ucReq)
	if err != nil {
		return errors.HandleDomainError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    convertTemplateToDTO(result.Template),
		"message": "Template created successfully",
	})
}

// PublishTemplate handles POST /api/tenants/{tenantId}/templates/{templateId}/publish
func (h *TemplateHandler) PublishTemplate(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Extract template ID from URL
	templateIDStr := c.Params("templateId")
	templateUUID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid template ID format", err)
	}
	templateID := shared.TemplateID(templateUUID)

	// Extract current user from context
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	var req PublishTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional for publish, ignore parse errors
		req = PublishTemplateRequest{}
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Convert to use case request
	ucReq := templateUC.PublishTemplateRequest{
		TemplateID:   templateID,
		TenantID:     tenantID,
		PublishedBy:  shared.UserID(currentUserID),
		ReleaseNotes: req.ReleaseNotes,
	}

	// Execute use case
	if err := h.publishTemplateUC.Execute(c.Context(), ucReq); err != nil {
		return errors.HandleDomainError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Template published successfully",
	})
}

// InstallTemplate handles POST /api/tenants/{tenantId}/templates/{templateId}/install
func (h *TemplateHandler) InstallTemplate(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Extract template ID from URL
	templateIDStr := c.Params("templateId")
	templateUUID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid template ID format", err)
	}
	templateID := shared.TemplateID(templateUUID)

	// Extract current user from context
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	var req InstallTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		// Configuration is optional, ignore parse errors
		req = InstallTemplateRequest{}
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Convert to use case request
	ucReq := templateUC.InstallTemplateRequest{
		TemplateID:    templateID,
		TenantID:      tenantID,
		InstalledBy:   shared.UserID(currentUserID),
		Configuration: req.Configuration,
	}

	// Execute use case
	result, err := h.installTemplateUC.Execute(c.Context(), ucReq)
	if err != nil {
		return errors.HandleDomainError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    convertInstallationToDTO(result.Installation),
		"message": "Template installed successfully",
	})
}

// SearchTemplates handles GET /api/templates/search
func (h *TemplateHandler) SearchTemplates(c *fiber.Ctx) error {
	// Extract current user from context
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Extract tenant from context
	tenantUUID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "Tenant context not found", nil)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Parse query parameters
	query := c.Query("q")
	categoryStr := c.Query("category")
	visibilityStr := c.Query("visibility")
	pricingModelStr := c.Query("pricing_model")
	tagsStr := c.Query("tags")

	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := c.QueryInt("page_size", 20)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Convert to use case request
	ucReq := templateUC.SearchTemplatesRequest{
		TenantID:    tenantID,
		RequestedBy: shared.UserID(currentUserID),
		Query:       query,
		Page:        page,
		PageSize:    pageSize,
	}

	// Parse optional filters
	if categoryStr != "" {
		category, err := template.ParseCategory(categoryStr)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid category", err)
		}
		ucReq.Category = &category
	}

	if visibilityStr != "" {
		visibility, err := template.ParseVisibility(visibilityStr)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid visibility", err)
		}
		ucReq.Visibility = &visibility
	}

	if pricingModelStr != "" {
		pricingModel, err := template.ParsePricingModel(pricingModelStr)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid pricing model", err)
		}
		ucReq.PricingModel = &pricingModel
	}

	if tagsStr != "" {
		// Simple comma-separated tags parsing
		tags := make([]string, 0)
		for _, tag := range strings.Split(tagsStr, ",") {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
		ucReq.Tags = tags
	}

	// Execute use case
	result, err := h.searchTemplatesUC.Execute(c.Context(), ucReq)
	if err != nil {
		return errors.HandleDomainError(err)
	}

	// Convert templates to DTOs
	templateDTOs := make([]TemplateDTO, len(result.Templates))
	for i, tmpl := range result.Templates {
		templateDTOs[i] = convertTemplateToDTO(tmpl)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"templates":    templateDTOs,
			"total_count":  result.TotalCount,
			"current_page": result.CurrentPage,
			"page_size":    result.PageSize,
			"has_next":     result.HasNext,
		},
		"message": "Templates retrieved successfully",
	})
}

// convertTemplateToDTO converts domain template to DTO
func convertTemplateToDTO(t *template.Template) TemplateDTO {
	dto := TemplateDTO{
		ID:            t.ID.String(),
		TenantID:      t.TenantID.String(),
		CreatedBy:     t.CreatedBy.String(),
		Name:          t.Name,
		Description:   t.Description,
		Category:      string(t.Category),
		Tags:          t.Tags,
		Configuration: t.Configuration,
		Variables:     t.Variables,
		Version:       t.Version.String(),
		Status:        string(t.Status),
		Visibility:    string(t.Visibility),
		PricingModel:  string(t.PricingModel),
		InstallCount:  t.InstallCount,
		Rating:        t.Rating,
		RatingCount:   t.RatingCount,
		CreatedAt:     t.CreatedAt.Time().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     t.UpdatedAt.Time().Format("2006-01-02T15:04:05Z07:00"),
	}

	if t.PriceAmount != nil {
		amount := t.PriceAmount.Amount()
		currency := t.PriceAmount.Currency()
		dto.PriceAmount = &amount
		dto.PriceCurrency = &currency
	}

	if t.PublishedAt != nil {
		published := t.PublishedAt.Time().Format("2006-01-02T15:04:05Z07:00")
		dto.PublishedAt = &published
	}

	return dto
}

// convertInstallationToDTO converts domain installation to DTO
func convertInstallationToDTO(i *template.Installation) InstallationDTO {
	return InstallationDTO{
		ID:            i.ID.String(),
		TemplateID:    i.TemplateID.String(),
		TenantID:      i.TenantID.String(),
		InstalledBy:   i.InstalledBy.String(),
		Configuration: i.Configuration,
		Status:        string(i.Status),
		InstalledAt:   i.CreatedAt.Time().Format("2006-01-02T15:04:05Z07:00"),
	}
}
