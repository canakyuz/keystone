package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/shared"
	"strings"
	"time"

	tenantdomain "nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/ports/events"
	"nexspaces-api/internal/core/ports/repositories"
	"nexspaces-api/internal/shared/errors"
)

// TenantUseCase handles tenant-related business logic
type TenantUseCase struct {
	tenantRepo repositories.TenantRepository
	eventBus   events.EventBus
}

// NewTenantUseCase creates a new tenant use case
func NewTenantUseCase(
	tenantRepo repositories.TenantRepository,
	eventBus events.EventBus,
) *TenantUseCase {
	return &TenantUseCase{
		tenantRepo: tenantRepo,
		eventBus:   eventBus,
	}
}

// CreateTenant creates a new tenant
func (uc *TenantUseCase) CreateTenant(ctx context.Context, req tenantdomain.CreateTenantRequest) (*tenantdomain.Tenant, error) {
	// Validate slug uniqueness
	available, err := uc.tenantRepo.IsSlugAvailable(ctx, req.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug availability: %w", err)
	}
	if !available {
		return nil, errors.NewValidationError("Slug is already taken", fmt.Errorf("slug: %s", req.Slug))
	}

	// Validate custom domain if provided
	if req.CustomDomain != nil {
		available, err := uc.tenantRepo.IsDomainAvailable(ctx, *req.CustomDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to check domain availability: %w", err)
		}
		if !available {
			return nil, errors.NewValidationError("Custom domain is already taken", fmt.Errorf("domain: %s", *req.CustomDomain))
		}
	}

	// Create tenant
	newTenant := tenantdomain.NewTenant(req)

	if err := uc.tenantRepo.Create(ctx, newTenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Publish domain event
	event := &events.TenantCreatedEvent{
		BaseEvent: events.BaseEvent{
			ID:          uuid.New().String(),
			Type:        events.EventTypeTenantCreated,
			AggregateID: newTenant.ID.String(),
			TenantID:    newTenant.ID.String(),
			OccurredAt:  time.Now(),
			Payload: map[string]interface{}{
				"tenant_name": newTenant.Name,
				"tenant_slug": newTenant.Slug,
			},
		},
		TenantName: newTenant.Name,
		TenantSlug: newTenant.Slug.String(),
	}

	if err := uc.eventBus.Publish(ctx, event); err != nil {
		// Log error but don't fail the operation
		// In production, you might want to use a proper logger
		fmt.Printf("Failed to publish tenant created event: %v\n", err)
	}

	return newTenant, nil
}

// GetTenant retrieves a tenant by ID
func (uc *TenantUseCase) GetTenant(ctx context.Context, tenantID uuid.UUID) (*tenantdomain.Tenant, error) {
	tenant, err := uc.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug
func (uc *TenantUseCase) GetTenantBySlug(ctx context.Context, slug string) (*tenantdomain.Tenant, error) {
	tenant, err := uc.tenantRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	if tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	return tenant, nil
}

// GetTenantByDomain retrieves a tenant by custom domain
func (uc *TenantUseCase) GetTenantByDomain(ctx context.Context, domain string) (*tenantdomain.Tenant, error) {
	tenant, err := uc.tenantRepo.GetByCustomDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by domain: %w", err)
	}

	if tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	return tenant, nil
}

// UpdateTenant updates an existing tenant
func (uc *TenantUseCase) UpdateTenant(ctx context.Context, tenantID uuid.UUID, req tenantdomain.UpdateTenantRequest) (*tenantdomain.Tenant, error) {
	// Get existing tenant
	existingTenant, err := uc.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if existingTenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// Track changes for event
	changes := make(map[string]interface{})

	// Update fields
	if req.Name != nil && *req.Name != existingTenant.Name {
		changes["name"] = map[string]string{
			"old": existingTenant.Name,
			"new": *req.Name,
		}
		existingTenant.Name = *req.Name
	}

	if req.CustomDomain != nil {
		oldDomain := ""
		if existingTenant.CustomDomain != nil {
			oldDomain = existingTenant.CustomDomain.String()
		}

		if *req.CustomDomain != oldDomain {
			// Validate domain availability
			available, err := uc.tenantRepo.IsDomainAvailable(ctx, *req.CustomDomain)
			if err != nil {
				return nil, fmt.Errorf("failed to check domain availability: %w", err)
			}
			if !available {
				return nil, errors.NewValidationError("Custom domain is already taken", fmt.Errorf("domain: %s", *req.CustomDomain))
			}

			changes["custom_domain"] = map[string]string{
				"old": oldDomain,
				"new": *req.CustomDomain,
			}

			// Create new domain value object
			newDomain, err := shared.NewDomain(*req.CustomDomain)
			if err != nil {
				return nil, fmt.Errorf("invalid domain: %w", err)
			}
			existingTenant.CustomDomain = newDomain
		}
	}

	if req.Status != nil && *req.Status != existingTenant.Status {
		changes["status"] = map[string]tenantdomain.Status{
			"old": existingTenant.Status,
			"new": *req.Status,
		}
		existingTenant.Status = *req.Status
	}

	if req.Settings != nil {
		// Convert settings struct to map
		var newSettings map[string]interface{}
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal settings: %w", err)
		}
		if err := json.Unmarshal(settingsJSON, &newSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}

		changes["settings"] = map[string]interface{}{
			"old": existingTenant.Settings,
			"new": newSettings,
		}
		existingTenant.Settings = newSettings
	}

	existingTenant.UpdatedAt = shared.NewTimestamp()

	// Save changes
	if err := uc.tenantRepo.Update(ctx, existingTenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Publish domain event if there were changes
	if len(changes) > 0 {
		event := &events.TenantUpdatedEvent{
			BaseEvent: events.BaseEvent{
				ID:          uuid.New().String(),
				Type:        events.EventTypeTenantUpdated,
				AggregateID: existingTenant.ID.String(),
				TenantID:    existingTenant.ID.String(),
				OccurredAt:  time.Now(),
				Payload: map[string]interface{}{
					"changes": changes,
				},
			},
			Changes: changes,
		}

		if err := uc.eventBus.Publish(ctx, event); err != nil {
			fmt.Printf("Failed to publish tenant updated event: %v\n", err)
		}
	}

	return existingTenant, nil
}

// ListTenants retrieves tenants with pagination
func (uc *TenantUseCase) ListTenants(ctx context.Context, limit, offset int) ([]*tenantdomain.Tenant, int, error) {
	criteria := repositories.TenantListCriteria{
		Page:     offset/limit + 1,
		PageSize: limit,
	}

	tenants, total, err := uc.tenantRepo.List(ctx, criteria)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}

	return tenants, total, nil
}

// SearchTenants searches tenants by name or slug
func (uc *TenantUseCase) SearchTenants(ctx context.Context, query string, limit, offset int) ([]*tenantdomain.Tenant, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*tenantdomain.Tenant{}, nil
	}

	tenants, err := uc.tenantRepo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search tenants: %w", err)
	}

	return tenants, nil
}

// ActivateTenant activates a tenant
func (uc *TenantUseCase) ActivateTenant(ctx context.Context, tenantID uuid.UUID) error {
	err := uc.tenantRepo.ActivateTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to activate tenant: %w", err)
	}

	return nil
}

// DeactivateTenant deactivates a tenant
func (uc *TenantUseCase) DeactivateTenant(ctx context.Context, tenantID uuid.UUID) error {
	err := uc.tenantRepo.DeactivateTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to deactivate tenant: %w", err)
	}

	return nil
}

// DeleteTenant soft deletes a tenant
func (uc *TenantUseCase) DeleteTenant(ctx context.Context, tenantID uuid.UUID) error {
	// Get tenant to verify it exists
	existingTenant, err := uc.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if existingTenant == nil {
		return fmt.Errorf("tenant not found")
	}

	// Soft delete the tenant
	if err := uc.tenantRepo.Delete(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// IsSlugAvailable checks if a slug is available
func (uc *TenantUseCase) IsSlugAvailable(ctx context.Context, slug string) (bool, error) {
	available, err := uc.tenantRepo.IsSlugAvailable(ctx, slug)
	if err != nil {
		return false, fmt.Errorf("failed to check slug availability: %w", err)
	}

	return available, nil
}

// IsDomainAvailable checks if a custom domain is available
func (uc *TenantUseCase) IsDomainAvailable(ctx context.Context, domain string) (bool, error) {
	available, err := uc.tenantRepo.IsDomainAvailable(ctx, domain)
	if err != nil {
		return false, fmt.Errorf("failed to check domain availability: %w", err)
	}

	return available, nil
}

// GetTenantStats returns tenant statistics
func (uc *TenantUseCase) GetTenantStats(ctx context.Context) (*TenantStats, error) {
	totalTenants, err := uc.tenantRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total tenant count: %w", err)
	}

	activeTenants, err := uc.tenantRepo.GetActiveTenantsCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tenant count: %w", err)
	}

	return &TenantStats{
		TotalTenants:    totalTenants,
		ActiveTenants:   activeTenants,
		InactiveTenants: totalTenants - activeTenants,
	}, nil
}

// TenantStats represents tenant statistics
type TenantStats struct {
	TotalTenants    int `json:"total_tenants"`
	ActiveTenants   int `json:"active_tenants"`
	InactiveTenants int `json:"inactive_tenants"`
}
