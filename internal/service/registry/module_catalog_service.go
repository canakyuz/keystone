package registry

import (
	"context"
	"fmt"

	"nexpaces-api/internal/domain/registry"
)

// ModuleCatalogService handles module catalog and marketplace operations
type ModuleCatalogService struct {
	moduleRepo registry.ModuleRepository
}

// NewModuleCatalogService creates a new module catalog service
func NewModuleCatalogService(moduleRepo registry.ModuleRepository) *ModuleCatalogService {
	return &ModuleCatalogService{
		moduleRepo: moduleRepo,
	}
}

// ListPublicModules retrieves all public modules for marketplace
func (s *ModuleCatalogService) ListPublicModules(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	// Ensure we only return public, active modules
	filters.IsPublic = boolPtr(true)
	filters.Status = moduleStatusPtr(registry.ModuleStatusActive)

	modules, err := s.moduleRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list public modules: %w", err)
	}

	return modules, nil
}

// SearchModules searches modules in the marketplace
func (s *ModuleCatalogService) SearchModules(ctx context.Context, query string, filters registry.ModuleFilters) ([]*registry.Module, error) {
	// Ensure we only search public, active modules
	filters.IsPublic = boolPtr(true)
	filters.Status = moduleStatusPtr(registry.ModuleStatusActive)

	modules, err := s.moduleRepo.Search(ctx, query, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search modules: %w", err)
	}

	return modules, nil
}

// GetModuleByID retrieves a module by ID
func (s *ModuleCatalogService) GetModuleByID(ctx context.Context, id string) (*registry.Module, error) {
	module, err := s.moduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get module by ID: %w", err)
	}

	return module, nil
}

// GetModuleByCode retrieves a module by code
func (s *ModuleCatalogService) GetModuleByCode(ctx context.Context, code string) (*registry.Module, error) {
	module, err := s.moduleRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get module by code: %w", err)
	}

	return module, nil
}

// GetModuleBySlug retrieves a module by slug
func (s *ModuleCatalogService) GetModuleBySlug(ctx context.Context, slug string) (*registry.Module, error) {
	module, err := s.moduleRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get module by slug: %w", err)
	}

	return module, nil
}

// GetPopularModules retrieves popular modules sorted by install count
func (s *ModuleCatalogService) GetPopularModules(ctx context.Context, limit int) ([]*registry.Module, error) {
	filters := registry.ModuleFilters{
		IsPublic:  boolPtr(true),
		Status:    moduleStatusPtr(registry.ModuleStatusActive),
		SortBy:    "install_count",
		SortOrder: "desc",
		Limit:     limit,
		Offset:    0,
	}

	modules, err := s.moduleRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular modules: %w", err)
	}

	return modules, nil
}

// GetTopRatedModules retrieves top rated modules
func (s *ModuleCatalogService) GetTopRatedModules(ctx context.Context, limit int) ([]*registry.Module, error) {
	filters := registry.ModuleFilters{
		IsPublic:  boolPtr(true),
		Status:    moduleStatusPtr(registry.ModuleStatusActive),
		SortBy:    "rating",
		SortOrder: "desc",
		Limit:     limit,
		Offset:    0,
	}

	modules, err := s.moduleRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get top rated modules: %w", err)
	}

	return modules, nil
}

// GetModulesByCategory retrieves modules by category
func (s *ModuleCatalogService) GetModulesByCategory(ctx context.Context, category registry.ModuleCategory, filters registry.ModuleFilters) ([]*registry.Module, error) {
	filters.Category = &category
	filters.IsPublic = boolPtr(true)
	filters.Status = moduleStatusPtr(registry.ModuleStatusActive)

	modules, err := s.moduleRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get modules by category: %w", err)
	}

	return modules, nil
}

// GetModulesByPricingModel retrieves modules by pricing model
func (s *ModuleCatalogService) GetModulesByPricingModel(ctx context.Context, pricingModel registry.PricingModel, filters registry.ModuleFilters) ([]*registry.Module, error) {
	filters.PricingModel = &pricingModel
	filters.IsPublic = boolPtr(true)
	filters.Status = moduleStatusPtr(registry.ModuleStatusActive)

	modules, err := s.moduleRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get modules by pricing model: %w", err)
	}

	return modules, nil
}

// GetFreeModules retrieves all free modules
func (s *ModuleCatalogService) GetFreeModules(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	pricingModel := registry.PricingFree
	return s.GetModulesByPricingModel(ctx, pricingModel, filters)
}

// GetNewModules retrieves recently added modules
func (s *ModuleCatalogService) GetNewModules(ctx context.Context, limit int) ([]*registry.Module, error) {
	filters := registry.ModuleFilters{
		IsPublic:  boolPtr(true),
		Status:    moduleStatusPtr(registry.ModuleStatusActive),
		SortBy:    "created_at",
		SortOrder: "desc",
		Limit:     limit,
		Offset:    0,
	}

	modules, err := s.moduleRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get new modules: %w", err)
	}

	return modules, nil
}

// GetBetaModules retrieves beta modules
func (s *ModuleCatalogService) GetBetaModules(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	filters.IsBeta = boolPtr(true)
	filters.IsPublic = boolPtr(true)
	filters.Status = moduleStatusPtr(registry.ModuleStatusActive)

	modules, err := s.moduleRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get beta modules: %w", err)
	}

	return modules, nil
}

// GetModuleInstallCount retrieves install count for a module
func (s *ModuleCatalogService) GetModuleInstallCount(ctx context.Context, moduleID string) (int, error) {
	count, err := s.moduleRepo.GetInstallCount(ctx, moduleID)
	if err != nil {
		return 0, fmt.Errorf("failed to get module install count: %w", err)
	}

	return count, nil
}

// ValidateModuleExists checks if a module exists and is accessible
func (s *ModuleCatalogService) ValidateModuleExists(ctx context.Context, moduleID string) error {
	_, err := s.moduleRepo.GetByID(ctx, moduleID)
	if err != nil {
		return fmt.Errorf("module validation failed: %w", err)
	}

	return nil
}

// IsModulePublic checks if a module is public and active
func (s *ModuleCatalogService) IsModulePublic(ctx context.Context, moduleID string) (bool, error) {
	module, err := s.moduleRepo.GetByID(ctx, moduleID)
	if err != nil {
		return false, fmt.Errorf("failed to check module public status: %w", err)
	}

	return module.IsPubliclyAvailable(), nil
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func moduleStatusPtr(s registry.ModuleStatus) *registry.ModuleStatus {
	return &s
}
