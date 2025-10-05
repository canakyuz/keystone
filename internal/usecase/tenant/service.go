package tenant

import (
	"context"
	"fmt"

	"nexpaces-api/internal/domain/tenant"
	tenantRepo "nexpaces-api/internal/repository/tenant"
	"nexpaces-api/pkg/logger"
	"nexpaces-api/pkg/validator"
)

// Service handles tenant business logic
type Service struct {
	repo        tenantRepo.Repository
	validator   *validator.Validator
	logger      *logger.Logger
	provisioner *ProvisioningService
}

// NewService creates a new tenant service
func NewService(repo tenantRepo.Repository, val *validator.Validator, log *logger.Logger, provisioner *ProvisioningService) *Service {
	return &Service{
		repo:        repo,
		validator:   val,
		logger:      log,
		provisioner: provisioner,
	}
}

// Create creates a new tenant
func (s *Service) Create(ctx context.Context, req *CreateTenantRequest) (*TenantResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Check if slug already exists
	exists, err := s.repo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to check tenant slug existence")
		return nil, fmt.Errorf("failed to check tenant existence: %w", err)
	}
	if exists {
		return nil, tenant.ErrTenantSlugTaken
	}

	// Check if email already exists
	exists, err = s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to check tenant email existence")
		return nil, fmt.Errorf("failed to check tenant existence: %w", err)
	}
	if exists {
		return nil, tenant.ErrTenantEmailTaken
	}

	// Create tenant domain entity
	t, err := tenant.New(req.Name, req.Slug, req.Email, tenant.SubscriptionPlan(req.Plan))
	if err != nil {
		return nil, err
	}

	if s.provisioner == nil {
		return nil, fmt.Errorf("tenant provisioning service is not configured")
	}

	schemaName, err := s.provisioner.GenerateSchemaName(ctx, req.Slug)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to generate tenant schema name")
		return nil, fmt.Errorf("failed to generate tenant schema: %w", err)
	}

	if err := t.SetSchemaName(schemaName); err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repo.Create(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to create tenant")
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	if err := s.provisioner.ProvisionTenantSchema(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to provision tenant schema")
		_ = s.repo.Delete(ctx, t.ID)
		return nil, fmt.Errorf("failed to provision tenant schema: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
		"slug":      t.Slug,
		"plan":      t.Plan,
	}).Info("Tenant created successfully")

	return ToResponse(t), nil
}

// GetByID retrieves a tenant by ID
func (s *Service) GetByID(ctx context.Context, id string) (*TenantResponse, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToResponse(t), nil
}

// GetBySlug retrieves a tenant by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*TenantResponse, error) {
	t, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return ToResponse(t), nil
}

// GetByEmail retrieves a tenant by email
func (s *Service) GetByEmail(ctx context.Context, email string) (*TenantResponse, error) {
	t, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return ToResponse(t), nil
}

// List retrieves all tenants with pagination
func (s *Service) List(ctx context.Context, page, perPage int, status, plan, search string) (*TenantListResponse, error) {
	// Set defaults
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Build filters
	filters := tenantRepo.ListFilters{
		Limit:     perPage,
		Offset:    (page - 1) * perPage,
		Search:    search,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	if status != "" {
		s := tenant.TenantStatus(status)
		filters.Status = &s
	}

	if plan != "" {
		p := tenant.SubscriptionPlan(plan)
		filters.Plan = &p
	}

	// Fetch from repository
	tenants, total, err := s.repo.List(ctx, filters)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to list tenants")
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	return ToResponseList(tenants, total, page, perPage), nil
}

// Update updates a tenant
func (s *Service) Update(ctx context.Context, id string, req *UpdateTenantRequest) (*TenantResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Get existing tenant
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != "" {
		t.Name = req.Name
	}
	if req.Email != "" {
		// Check if new email is already taken
		if req.Email != t.Email {
			exists, err := s.repo.ExistsByEmail(ctx, req.Email)
			if err != nil {
				s.logger.ErrorWithErr(err, "failed to check email existence")
				return nil, fmt.Errorf("failed to check email: %w", err)
			}
			if exists {
				return nil, tenant.ErrTenantEmailTaken
			}
		}
		t.Email = req.Email
	}
	if req.Phone != "" {
		t.Phone = req.Phone
	}

	// Save to repository
	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to update tenant")
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
	}).Info("Tenant updated successfully")

	return ToResponse(t), nil
}

// Suspend suspends a tenant account
func (s *Service) Suspend(ctx context.Context, id, reason string) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := t.Suspend(reason); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to suspend tenant")
		return fmt.Errorf("failed to suspend tenant: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
		"reason":    reason,
	}).Warn("Tenant suspended")

	return nil
}

// Activate activates a tenant account
func (s *Service) Activate(ctx context.Context, id string) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := t.Activate(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to activate tenant")
		return fmt.Errorf("failed to activate tenant: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
	}).Info("Tenant activated")

	return nil
}

// UpgradePlan upgrades tenant subscription plan
func (s *Service) UpgradePlan(ctx context.Context, id string, req *UpgradePlanRequest) (*TenantResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := t.UpgradePlan(tenant.SubscriptionPlan(req.Plan)); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to upgrade plan")
		return nil, fmt.Errorf("failed to upgrade plan: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
		"new_plan":  req.Plan,
	}).Info("Tenant plan upgraded")

	return ToResponse(t), nil
}

// SetCustomDomain sets custom domain for tenant
func (s *Service) SetCustomDomain(ctx context.Context, id string, req *SetCustomDomainRequest) (*TenantResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Check if domain is already taken
	exists, err := s.repo.ExistsByCustomDomain(ctx, req.Domain)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to check domain existence")
		return nil, fmt.Errorf("failed to check domain: %w", err)
	}
	if exists {
		return nil, tenant.ErrCustomDomainTaken
	}

	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := t.SetCustomDomain(req.Domain); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to set custom domain")
		return nil, fmt.Errorf("failed to set custom domain: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
		"domain":    req.Domain,
	}).Info("Custom domain set for tenant")

	return ToResponse(t), nil
}

// VerifyCustomDomain verifies tenant custom domain
func (s *Service) VerifyCustomDomain(ctx context.Context, id string) (*TenantResponse, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := t.VerifyCustomDomain(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to verify custom domain")
		return nil, fmt.Errorf("failed to verify custom domain: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
		"domain":    t.CustomDomain,
	}).Info("Custom domain verified")

	return ToResponse(t), nil
}

// Delete soft deletes a tenant
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "failed to delete tenant")
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": id,
	}).Warn("Tenant deleted")

	return nil
}

// UpdateBranding updates tenant branding settings
func (s *Service) UpdateBranding(ctx context.Context, id string, req *UpdateBrandingRequest) (*TenantResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Get existing tenant
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get current branding settings or create new
	var branding map[string]interface{}
	if existingBranding, exists := t.Settings["branding"]; exists {
		if brandingMap, ok := existingBranding.(map[string]interface{}); ok {
			branding = brandingMap
		} else {
			branding = make(map[string]interface{})
		}
	} else {
		branding = make(map[string]interface{})
	}

	// Update only provided fields
	if req.Logo != nil {
		branding["logo"] = *req.Logo
	}
	if req.Favicon != nil {
		branding["favicon"] = *req.Favicon
	}
	if req.PrimaryColor != nil {
		branding["primary_color"] = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		branding["secondary_color"] = *req.SecondaryColor
	}
	if req.AccentColor != nil {
		branding["accent_color"] = *req.AccentColor
	}
	if req.FontFamily != nil {
		branding["font_family"] = *req.FontFamily
	}
	if req.CustomCSS != nil {
		branding["custom_css"] = *req.CustomCSS
	}

	// Update tenant settings
	t.UpdateSettings("branding", branding)

	// Save to repository
	if err := s.repo.Update(ctx, t); err != nil {
		s.logger.ErrorWithErr(err, "failed to update branding")
		return nil, fmt.Errorf("failed to update branding: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id": t.ID,
	}).Info("Tenant branding updated successfully")

	return ToResponse(t), nil
}

// GetStats retrieves tenant statistics
func (s *Service) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count by status
	for _, status := range []tenant.TenantStatus{
		tenant.TenantStatusActive,
		tenant.TenantStatusSuspended,
		tenant.TenantStatusInactive,
		tenant.TenantStatusTrial,
	} {
		count, err := s.repo.CountByStatus(ctx, status)
		if err != nil {
			s.logger.ErrorWithErr(err, "failed to count tenants by status")
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}
		stats[string(status)] = count
	}

	// Count by plan
	for _, plan := range []tenant.SubscriptionPlan{
		tenant.PlanFree,
		tenant.PlanStarter,
		tenant.PlanPro,
		tenant.PlanEnterprise,
	} {
		count, err := s.repo.CountByPlan(ctx, plan)
		if err != nil {
			s.logger.ErrorWithErr(err, "failed to count tenants by plan")
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}
		stats[string(plan)] = count
	}

	return stats, nil
}
