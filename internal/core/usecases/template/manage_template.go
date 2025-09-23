package template

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/template"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
	"nexspaces-api/internal/core/ports/events"
	"nexspaces-api/internal/core/ports/repositories"
	"nexspaces-api/internal/core/ports/services"
)

// CreateTemplateUseCase handles template creation with proper validation and security
type CreateTemplateUseCase struct {
	templateRepo repositories.TemplateRepository
	userRepo     repositories.UserRepository
	tenantRepo   repositories.TenantRepository
	eventBus     events.EventBus
}

func NewCreateTemplateUseCase(
	templateRepo repositories.TemplateRepository,
	userRepo repositories.UserRepository,
	tenantRepo repositories.TenantRepository,
	eventBus events.EventBus,
) *CreateTemplateUseCase {
	return &CreateTemplateUseCase{
		templateRepo: templateRepo,
		userRepo:     userRepo,
		tenantRepo:   tenantRepo,
		eventBus:     eventBus,
	}
}

type CreateTemplateRequest struct {
	TenantID      shared.TenantID        `json:"tenant_id" validate:"required"`
	CreatedBy     shared.UserID          `json:"created_by" validate:"required"`
	Name          string                 `json:"name" validate:"required,min=3,max=100"`
	Description   string                 `json:"description" validate:"required,min=10,max=1000"`
	Category      template.Category      `json:"category" validate:"required"`
	Tags          []string               `json:"tags,omitempty" validate:"max=10"`
	Configuration map[string]interface{} `json:"configuration" validate:"required"`
	Visibility    template.Visibility    `json:"visibility" validate:"required"`
	PricingModel  template.PricingModel  `json:"pricing_model" validate:"required"`
}

type CreateTemplateResponse struct {
	Template *template.Template `json:"template"`
}

func (uc *CreateTemplateUseCase) Execute(ctx context.Context, req CreateTemplateRequest) (*CreateTemplateResponse, error) {
	// Validate creator exists and has permissions
	creator, err := uc.userRepo.GetByID(ctx, uuid.UUID(req.TenantID), uuid.UUID(req.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("failed to get creator: %w", err)
	}

	// Verify tenant isolation
	if creator.TenantID != uuid.UUID(req.TenantID) {
		return nil, shared.ErrCrossTenantAccess
	}

	// Check if user can create templates
	if !creator.CanCreateTemplates() {
		return nil, shared.ErrInsufficientPermissions
	}

	// Validate tenant is active
	tenantEntity, err := uc.tenantRepo.GetByID(ctx, uuid.UUID(req.TenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenantEntity.Status != tenant.StatusActive {
		return nil, shared.ErrTenantNotActive
	}

	// Check if template name exists in tenant
	existing, err := uc.templateRepo.GetByNameAndTenant(ctx, req.Name, req.TenantID)
	if err != nil && !errors.Is(err, shared.ErrTemplateNotFound) {
		return nil, fmt.Errorf("failed to check existing template: %w", err)
	}

	if existing != nil {
		return nil, shared.ErrTemplateAlreadyExists
	}

	// Create template entity
	templateEntity, err := template.NewTemplate(
		req.TenantID,
		req.CreatedBy,
		req.Name,
		req.Description,
		req.Category,
		req.Tags,
		req.Configuration,
		req.Visibility,
		req.PricingModel,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Save to repository
	if err := uc.templateRepo.Create(ctx, templateEntity); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	// Publish domain event
	if err := uc.eventBus.Publish(ctx, &events.TemplateCreatedEvent{
		BaseEvent: events.BaseEvent{
			ID:          uuid.New().String(),
			Type:        events.EventTypeTemplateCreated,
			AggregateID: uuid.UUID(templateEntity.ID).String(),
			TenantID:    uuid.UUID(req.TenantID).String(),
			OccurredAt:  time.Now(),
		},
		TemplateID:   uuid.UUID(templateEntity.ID).String(),
		TemplateName: req.Name,
		Category:     string(req.Category),
	}); err != nil {
		// Log error but don't fail
	}

	return &CreateTemplateResponse{
		Template: templateEntity,
	}, nil
}

// PublishTemplateUseCase handles template publishing to marketplace
type PublishTemplateUseCase struct {
	templateRepo repositories.TemplateRepository
	userRepo     repositories.UserRepository
	eventBus     events.EventBus
}

func NewPublishTemplateUseCase(
	templateRepo repositories.TemplateRepository,
	userRepo repositories.UserRepository,
	eventBus events.EventBus,
) *PublishTemplateUseCase {
	return &PublishTemplateUseCase{
		templateRepo: templateRepo,
		userRepo:     userRepo,
		eventBus:     eventBus,
	}
}

type PublishTemplateRequest struct {
	TemplateID   shared.TemplateID `json:"template_id" validate:"required"`
	TenantID     shared.TenantID   `json:"tenant_id" validate:"required"`
	PublishedBy  shared.UserID     `json:"published_by" validate:"required"`
	ReleaseNotes string            `json:"release_notes,omitempty" validate:"max=500"`
}

func (uc *PublishTemplateUseCase) Execute(ctx context.Context, req PublishTemplateRequest) error {
	// Get template
	templateEntity, err := uc.templateRepo.GetByID(ctx, req.TemplateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Verify tenant isolation
	if templateEntity.TenantID != req.TenantID {
		return shared.ErrCrossTenantAccess
	}

	// Get publisher
	publisher, err := uc.userRepo.GetByID(ctx, req.TenantID, req.PublishedBy)
	if err != nil {
		return fmt.Errorf("failed to get publisher: %w", err)
	}

	// Verify publisher belongs to same tenant
	if publisher.TenantID != req.TenantID {
		return shared.ErrCrossTenantAccess
	}

	// Check permissions
	if !publisher.CanPublishTemplates() {
		return shared.ErrInsufficientPermissions
	}

	// Publish template
	if err := templateEntity.Publish(req.PublishedBy, req.ReleaseNotes); err != nil {
		return fmt.Errorf("failed to publish template: %w", err)
	}

	// Save changes
	if err := uc.templateRepo.Update(ctx, templateEntity); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, events.TemplatePublishedEvent{
		TemplateID:   templateEntity.ID,
		TenantID:     req.TenantID,
		PublishedBy:  req.PublishedBy,
		Version:      templateEntity.Version,
		ReleaseNotes: req.ReleaseNotes,
		PublishedAt:  templateEntity.UpdatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return nil
}

// InstallTemplateUseCase handles template installation for tenants
type InstallTemplateUseCase struct {
	templateRepo repositories.TemplateRepository
	userRepo     repositories.UserRepository
	tenantRepo   repositories.TenantRepository
	eventBus     events.EventBus
}

func NewInstallTemplateUseCase(
	templateRepo repositories.TemplateRepository,
	userRepo repositories.UserRepository,
	tenantRepo repositories.TenantRepository,
	eventBus events.EventBus,
) *InstallTemplateUseCase {
	return &InstallTemplateUseCase{
		templateRepo: templateRepo,
		userRepo:     userRepo,
		tenantRepo:   tenantRepo,
		eventBus:     eventBus,
	}
}

type InstallTemplateRequest struct {
	TemplateID    shared.TemplateID      `json:"template_id" validate:"required"`
	TenantID      shared.TenantID        `json:"tenant_id" validate:"required"`
	InstalledBy   shared.UserID          `json:"installed_by" validate:"required"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

type InstallTemplateResponse struct {
	Installation *template.Installation `json:"installation"`
}

func (uc *InstallTemplateUseCase) Execute(ctx context.Context, req InstallTemplateRequest) (*InstallTemplateResponse, error) {
	// Get template
	templateEntity, err := uc.templateRepo.GetByID(ctx, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Verify template is published or accessible
	if templateEntity.Status != template.StatusPublished {
		// Check if it's a private template in same tenant
		if templateEntity.TenantID != req.TenantID || templateEntity.Visibility != template.VisibilityPrivate {
			return nil, shared.ErrTemplateNotAccessible
		}
	}

	// Get installer
	installer, err := uc.userRepo.GetByID(ctx, req.TenantID, req.InstalledBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get installer: %w", err)
	}

	// Verify installer belongs to requesting tenant
	if installer.TenantID != req.TenantID {
		return nil, shared.ErrCrossTenantAccess
	}

	// Check permissions
	if !installer.CanInstallTemplates() {
		return nil, shared.ErrInsufficientPermissions
	}

	// Check if already installed
	existing, err := uc.templateRepo.GetInstallationByTemplateAndTenant(ctx, req.TemplateID, req.TenantID)
	if err != nil && !errors.Is(err, shared.ErrInstallationNotFound) {
		return nil, fmt.Errorf("failed to check existing installation: %w", err)
	}

	if existing != nil {
		return nil, shared.ErrTemplateAlreadyInstalled
	}

	// Create installation
	installation, err := template.NewInstallation(
		req.TemplateID,
		req.TenantID,
		req.InstalledBy,
		req.Configuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create installation: %w", err)
	}

	// Save installation
	if err := uc.templateRepo.CreateInstallation(ctx, installation); err != nil {
		return nil, fmt.Errorf("failed to save installation: %w", err)
	}

	// Update template install count
	if err := templateEntity.IncrementInstallCount(); err != nil {
		return nil, fmt.Errorf("failed to increment install count: %w", err)
	}

	if err := uc.templateRepo.Update(ctx, templateEntity); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, events.TemplateInstalledEvent{
		TemplateID:     req.TemplateID,
		TenantID:       req.TenantID,
		InstallationID: installation.ID,
		InstalledBy:    req.InstalledBy,
		InstalledAt:    installation.CreatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return &InstallTemplateResponse{
		Installation: installation,
	}, nil
}

// SearchTemplatesUseCase handles template marketplace search
type SearchTemplatesUseCase struct {
	templateRepo repositories.TemplateRepository
	userRepo     repositories.UserRepository
}

func NewSearchTemplatesUseCase(
	templateRepo repositories.TemplateRepository,
	userRepo repositories.UserRepository,
) *SearchTemplatesUseCase {
	return &SearchTemplatesUseCase{
		templateRepo: templateRepo,
		userRepo:     userRepo,
	}
}

type SearchTemplatesRequest struct {
	TenantID     shared.TenantID        `json:"tenant_id" validate:"required"`
	RequestedBy  shared.UserID          `json:"requested_by" validate:"required"`
	Query        string                 `json:"query,omitempty"`
	Category     *template.Category     `json:"category,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	PricingModel *template.PricingModel `json:"pricing_model,omitempty"`
	Visibility   *template.Visibility   `json:"visibility,omitempty"`
	Page         int                    `json:"page" validate:"min=1"`
	PageSize     int                    `json:"page_size" validate:"min=1,max=100"`
}

type SearchTemplatesResponse struct {
	Templates   []*template.Template `json:"templates"`
	TotalCount  int                  `json:"total_count"`
	CurrentPage int                  `json:"current_page"`
	PageSize    int                  `json:"page_size"`
	HasNext     bool                 `json:"has_next"`
}

func (uc *SearchTemplatesUseCase) Execute(ctx context.Context, req SearchTemplatesRequest) (*SearchTemplatesResponse, error) {
	// Validate requester
	requester, err := uc.userRepo.GetByID(ctx, req.TenantID, req.RequestedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester: %w", err)
	}

	if requester.TenantID != req.TenantID {
		return nil, shared.ErrCrossTenantAccess
	}

	// Build search criteria
	criteria := repositories.TemplateSearchCriteria{
		TenantID:     req.TenantID,
		Query:        req.Query,
		Category:     req.Category,
		Tags:         req.Tags,
		PricingModel: req.PricingModel,
		Visibility:   req.Visibility,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	// Search templates
	templates, totalCount, err := uc.templateRepo.Search(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search templates: %w", err)
	}

	// Calculate pagination
	hasNext := (req.Page * req.PageSize) < totalCount

	return &SearchTemplatesResponse{
		Templates:   templates,
		TotalCount:  totalCount,
		CurrentPage: req.Page,
		PageSize:    req.PageSize,
		HasNext:     hasNext,
	}, nil
}
