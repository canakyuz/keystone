package registry

import (
	"context"
	"fmt"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// ToolCatalogService handles tool catalog and marketplace operations
type ToolCatalogService struct {
	toolRepo registry.ToolRepository
}

// NewToolCatalogService creates a new tool catalog service
func NewToolCatalogService(toolRepo registry.ToolRepository) *ToolCatalogService {
	return &ToolCatalogService{
		toolRepo: toolRepo,
	}
}

// ListPublicTools retrieves all public tools for marketplace
func (s *ToolCatalogService) ListPublicTools(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	// Ensure we only return public, active tools
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)

	tools, err := s.toolRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list public tools: %w", err)
	}

	return tools, nil
}

// SearchTools searches tools in the marketplace
func (s *ToolCatalogService) SearchTools(ctx context.Context, query string, filters registry.ToolFilters) ([]*registry.Tool, error) {
	// Ensure we only search public, active tools
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)

	tools, err := s.toolRepo.Search(ctx, query, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search tools: %w", err)
	}

	return tools, nil
}

// GetToolByID retrieves a tool by ID
func (s *ToolCatalogService) GetToolByID(ctx context.Context, id string) (*registry.Tool, error) {
	tool, err := s.toolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool by ID: %w", err)
	}

	return tool, nil
}

// GetToolByCode retrieves a tool by code
func (s *ToolCatalogService) GetToolByCode(ctx context.Context, code string) (*registry.Tool, error) {
	tool, err := s.toolRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool by code: %w", err)
	}

	return tool, nil
}

// GetToolBySlug retrieves a tool by slug
func (s *ToolCatalogService) GetToolBySlug(ctx context.Context, slug string) (*registry.Tool, error) {
	tool, err := s.toolRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool by slug: %w", err)
	}

	return tool, nil
}

// GetPopularTools retrieves popular tools sorted by install count
func (s *ToolCatalogService) GetPopularTools(ctx context.Context, limit int) ([]*registry.Tool, error) {
	filters := registry.ToolFilters{
		IsPublic:  boolPtr(true),
		Status:    toolStatusPtr(registry.ToolStatusActive),
		SortBy:    "install_count",
		SortOrder: "desc",
		Limit:     limit,
		Offset:    0,
	}

	tools, err := s.toolRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular tools: %w", err)
	}

	return tools, nil
}

// GetTopRatedTools retrieves top rated tools
func (s *ToolCatalogService) GetTopRatedTools(ctx context.Context, limit int) ([]*registry.Tool, error) {
	filters := registry.ToolFilters{
		IsPublic:  boolPtr(true),
		Status:    toolStatusPtr(registry.ToolStatusActive),
		SortBy:    "rating",
		SortOrder: "desc",
		Limit:     limit,
		Offset:    0,
	}

	tools, err := s.toolRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get top rated tools: %w", err)
	}

	return tools, nil
}

// GetToolsByCategory retrieves tools by category
func (s *ToolCatalogService) GetToolsByCategory(ctx context.Context, category registry.ToolCategory, filters registry.ToolFilters) ([]*registry.Tool, error) {
	filters.Category = &category
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)

	tools, err := s.toolRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools by category: %w", err)
	}

	return tools, nil
}

// GetToolsByScope retrieves tools by scope
func (s *ToolCatalogService) GetToolsByScope(ctx context.Context, scope registry.ToolScope, filters registry.ToolFilters) ([]*registry.Tool, error) {
	filters.Scope = &scope
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)

	tools, err := s.toolRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools by scope: %w", err)
	}

	return tools, nil
}

// GetToolsByPricingModel retrieves tools by pricing model
func (s *ToolCatalogService) GetToolsByPricingModel(ctx context.Context, pricingModel registry.PricingModel, filters registry.ToolFilters) ([]*registry.Tool, error) {
	filters.PricingModel = &pricingModel
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)

	tools, err := s.toolRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools by pricing model: %w", err)
	}

	return tools, nil
}

// GetFreeTools retrieves all free tools
func (s *ToolCatalogService) GetFreeTools(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	pricingModel := registry.PricingFree
	return s.GetToolsByPricingModel(ctx, pricingModel, filters)
}

// GetPaymentTools retrieves payment-related tools
func (s *ToolCatalogService) GetPaymentTools(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	category := registry.ToolCategoryPayment
	return s.GetToolsByCategory(ctx, category, filters)
}

// GetIntegrationTools retrieves integration tools
func (s *ToolCatalogService) GetIntegrationTools(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	category := registry.ToolCategoryIntegration
	return s.GetToolsByCategory(ctx, category, filters)
}

// GetCommunicationTools retrieves communication tools
func (s *ToolCatalogService) GetCommunicationTools(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	category := registry.ToolCategoryCommunication
	return s.GetToolsByCategory(ctx, category, filters)
}

// GetNewTools retrieves recently added tools
func (s *ToolCatalogService) GetNewTools(ctx context.Context, limit int) ([]*registry.Tool, error) {
	filters := registry.ToolFilters{
		IsPublic:  boolPtr(true),
		Status:    toolStatusPtr(registry.ToolStatusActive),
		SortBy:    "created_at",
		SortOrder: "desc",
		Limit:     limit,
		Offset:    0,
	}

	tools, err := s.toolRepo.ListPublic(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get new tools: %w", err)
	}

	return tools, nil
}

// GetBetaTools retrieves beta tools
func (s *ToolCatalogService) GetBetaTools(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	filters.IsBeta = boolPtr(true)
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)

	tools, err := s.toolRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get beta tools: %w", err)
	}

	return tools, nil
}

// GetToolInstallCount retrieves install count for a tool
func (s *ToolCatalogService) GetToolInstallCount(ctx context.Context, toolID string) (int, error) {
	count, err := s.toolRepo.GetInstallCount(ctx, toolID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tool install count: %w", err)
	}

	return count, nil
}

// ValidateToolExists checks if a tool exists and is accessible
func (s *ToolCatalogService) ValidateToolExists(ctx context.Context, toolID string) error {
	_, err := s.toolRepo.GetByID(ctx, toolID)
	if err != nil {
		return fmt.Errorf("tool validation failed: %w", err)
	}

	return nil
}

// IsToolPublic checks if a tool is public and active
func (s *ToolCatalogService) IsToolPublic(ctx context.Context, toolID string) (bool, error) {
	tool, err := s.toolRepo.GetByID(ctx, toolID)
	if err != nil {
		return false, fmt.Errorf("failed to check tool public status: %w", err)
	}

	return tool.IsPubliclyAvailable(), nil
}

// RequiresAPIKeys checks if a tool requires API keys
func (s *ToolCatalogService) RequiresAPIKeys(ctx context.Context, toolID string) (bool, error) {
	tool, err := s.toolRepo.GetByID(ctx, toolID)
	if err != nil {
		return false, fmt.Errorf("failed to check API keys requirement: %w", err)
	}

	return tool.RequiresAPIKeys, nil
}

// GetToolTransactionFees retrieves transaction fee information
func (s *ToolCatalogService) GetToolTransactionFees(ctx context.Context, toolID string) (float64, float64, error) {
	tool, err := s.toolRepo.GetByID(ctx, toolID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get tool transaction fees: %w", err)
	}

	return tool.TransactionFeePercentage, tool.TransactionFeeFixed, nil
}

func toolStatusPtr(s registry.ToolStatus) *registry.ToolStatus {
	return &s
}
