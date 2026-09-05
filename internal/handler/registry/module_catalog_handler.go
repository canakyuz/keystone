package registry

import (
	"strconv"

	"github.com/canakyuz/keystone/internal/domain/registry"
	dto "github.com/canakyuz/keystone/internal/dto/registry"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
	"github.com/gofiber/fiber/v2"
)

// ModuleCatalogHandler handles module catalog HTTP requests
type ModuleCatalogHandler struct {
	catalogService *registryService.ModuleCatalogService
}

// NewModuleCatalogHandler creates a new module catalog handler
func NewModuleCatalogHandler(catalogService *registryService.ModuleCatalogService) *ModuleCatalogHandler {
	return &ModuleCatalogHandler{
		catalogService: catalogService,
	}
}

// ListPublicModules retrieves public modules for marketplace
// GET /api/v1/registry/modules
func (h *ModuleCatalogHandler) ListPublicModules(c *fiber.Ctx) error {
	var req dto.ModuleListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	// Build filters
	filters := buildModuleFilters(req)

	modules, err := h.catalogService.ListPublicModules(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"pagination": dto.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			TotalItems: len(response),
			TotalPages: (len(response) + req.PerPage - 1) / req.PerPage,
		},
	})
}

// SearchModules searches modules in marketplace
// GET /api/v1/registry/modules/search
func (h *ModuleCatalogHandler) SearchModules(c *fiber.Ctx) error {
	var req dto.ModuleSearchRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	// Build filters
	filters := registry.ModuleFilters{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PerPage,
		Offset:    (req.Page - 1) * req.PerPage,
	}

	if req.Category != "" {
		category := registry.ModuleCategory(req.Category)
		filters.Category = &category
	}

	if req.PricingModel != "" {
		pricingModel := registry.PricingModel(req.PricingModel)
		filters.PricingModel = &pricingModel
	}

	modules, err := h.catalogService.SearchModules(c.Context(), req.Query, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"pagination": dto.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			TotalItems: len(response),
			TotalPages: (len(response) + req.PerPage - 1) / req.PerPage,
		},
	})
}

// GetModuleByID retrieves a module by ID
// GET /api/v1/registry/modules/:id
func (h *ModuleCatalogHandler) GetModuleByID(c *fiber.Ctx) error {
	id := c.Params("id")

	module, err := h.catalogService.GetModuleByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Module not found",
		})
	}

	response := dto.ToModuleDetailResponse(module)

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetModuleBySlug retrieves a module by slug
// GET /api/v1/registry/modules/slug/:slug
func (h *ModuleCatalogHandler) GetModuleBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	module, err := h.catalogService.GetModuleBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Module not found",
		})
	}

	response := dto.ToModuleDetailResponse(module)

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetPopularModules retrieves popular modules
// GET /api/v1/registry/modules/popular
func (h *ModuleCatalogHandler) GetPopularModules(c *fiber.Ctx) error {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	modules, err := h.catalogService.GetPopularModules(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetTopRatedModules retrieves top rated modules
// GET /api/v1/registry/modules/top-rated
func (h *ModuleCatalogHandler) GetTopRatedModules(c *fiber.Ctx) error {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	modules, err := h.catalogService.GetTopRatedModules(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetModulesByCategory retrieves modules by category
// GET /api/v1/registry/modules/category/:category
func (h *ModuleCatalogHandler) GetModulesByCategory(c *fiber.Ctx) error {
	categoryParam := c.Params("category")
	category := registry.ModuleCategory(categoryParam)

	var req dto.ModuleListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	filters := buildModuleFilters(req)

	modules, err := h.catalogService.GetModulesByCategory(c.Context(), category, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"pagination": dto.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			TotalItems: len(response),
			TotalPages: (len(response) + req.PerPage - 1) / req.PerPage,
		},
	})
}

// GetFreeModules retrieves free modules
// GET /api/v1/registry/modules/free
func (h *ModuleCatalogHandler) GetFreeModules(c *fiber.Ctx) error {
	var req dto.ModuleListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	filters := buildModuleFilters(req)

	modules, err := h.catalogService.GetFreeModules(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"pagination": dto.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			TotalItems: len(response),
			TotalPages: (len(response) + req.PerPage - 1) / req.PerPage,
		},
	})
}

// GetNewModules retrieves recently added modules
// GET /api/v1/registry/modules/new
func (h *ModuleCatalogHandler) GetNewModules(c *fiber.Ctx) error {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	modules, err := h.catalogService.GetNewModules(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// Helper function to build module filters from request
func buildModuleFilters(req dto.ModuleListRequest) registry.ModuleFilters {
	filters := registry.ModuleFilters{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PerPage,
		Offset:    (req.Page - 1) * req.PerPage,
		IsPublic:  req.IsPublic,
		IsBeta:    req.IsBeta,
	}

	if req.Category != "" {
		category := registry.ModuleCategory(req.Category)
		filters.Category = &category
	}

	if req.ModuleType != "" {
		moduleType := registry.ModuleType(req.ModuleType)
		filters.ModuleType = &moduleType
	}

	if req.PricingModel != "" {
		pricingModel := registry.PricingModel(req.PricingModel)
		filters.PricingModel = &pricingModel
	}

	if req.Status != "" {
		status := registry.ModuleStatus(req.Status)
		filters.Status = &status
	}

	return filters
}
