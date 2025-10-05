package registry

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"nexpaces-api/internal/domain/registry"
	dto "nexpaces-api/internal/dto/registry"
	registryService "nexpaces-api/internal/service/registry"
)

// ToolCatalogHandler handles tool catalog HTTP requests
type ToolCatalogHandler struct {
	catalogService *registryService.ToolCatalogService
}

// NewToolCatalogHandler creates a new tool catalog handler
func NewToolCatalogHandler(catalogService *registryService.ToolCatalogService) *ToolCatalogHandler {
	return &ToolCatalogHandler{
		catalogService: catalogService,
	}
}

// ListPublicTools retrieves public tools for marketplace
// GET /api/v1/registry/tools
func (h *ToolCatalogHandler) ListPublicTools(c *fiber.Ctx) error {
	var req dto.ToolListRequest
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
	filters := buildToolFilters(req)

	tools, err := h.catalogService.ListPublicTools(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
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

// SearchTools searches tools in marketplace
// GET /api/v1/registry/tools/search
func (h *ToolCatalogHandler) SearchTools(c *fiber.Ctx) error {
	var req dto.ToolSearchRequest
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
	filters := registry.ToolFilters{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PerPage,
		Offset:    (req.Page - 1) * req.PerPage,
	}

	if req.Category != "" {
		category := registry.ToolCategory(req.Category)
		filters.Category = &category
	}

	if req.Scope != "" {
		scope := registry.ToolScope(req.Scope)
		filters.Scope = &scope
	}

	if req.PricingModel != "" {
		pricingModel := registry.PricingModel(req.PricingModel)
		filters.PricingModel = &pricingModel
	}

	tools, err := h.catalogService.SearchTools(c.Context(), req.Query, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
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

// GetToolByID retrieves a tool by ID
// GET /api/v1/registry/tools/:id
func (h *ToolCatalogHandler) GetToolByID(c *fiber.Ctx) error {
	id := c.Params("id")

	tool, err := h.catalogService.GetToolByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Tool not found",
		})
	}

	response := dto.ToToolDetailResponse(tool)

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetToolBySlug retrieves a tool by slug
// GET /api/v1/registry/tools/slug/:slug
func (h *ToolCatalogHandler) GetToolBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	tool, err := h.catalogService.GetToolBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Tool not found",
		})
	}

	response := dto.ToToolDetailResponse(tool)

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetPopularTools retrieves popular tools
// GET /api/v1/registry/tools/popular
func (h *ToolCatalogHandler) GetPopularTools(c *fiber.Ctx) error {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	tools, err := h.catalogService.GetPopularTools(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetTopRatedTools retrieves top rated tools
// GET /api/v1/registry/tools/top-rated
func (h *ToolCatalogHandler) GetTopRatedTools(c *fiber.Ctx) error {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	tools, err := h.catalogService.GetTopRatedTools(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetToolsByCategory retrieves tools by category
// GET /api/v1/registry/tools/category/:category
func (h *ToolCatalogHandler) GetToolsByCategory(c *fiber.Ctx) error {
	categoryParam := c.Params("category")
	category := registry.ToolCategory(categoryParam)

	var req dto.ToolListRequest
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

	filters := buildToolFilters(req)

	tools, err := h.catalogService.GetToolsByCategory(c.Context(), category, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
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

// GetFreeTools retrieves free tools
// GET /api/v1/registry/tools/free
func (h *ToolCatalogHandler) GetFreeTools(c *fiber.Ctx) error {
	var req dto.ToolListRequest
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

	filters := buildToolFilters(req)

	tools, err := h.catalogService.GetFreeTools(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
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

// GetNewTools retrieves recently added tools
// GET /api/v1/registry/tools/new
func (h *ToolCatalogHandler) GetNewTools(c *fiber.Ctx) error {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	tools, err := h.catalogService.GetNewTools(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// Helper function to build tool filters from request
func buildToolFilters(req dto.ToolListRequest) registry.ToolFilters {
	filters := registry.ToolFilters{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PerPage,
		Offset:    (req.Page - 1) * req.PerPage,
		IsPublic:  req.IsPublic,
		IsBeta:    req.IsBeta,
	}

	if req.Category != "" {
		category := registry.ToolCategory(req.Category)
		filters.Category = &category
	}

	if req.ToolType != "" {
		toolType := registry.ToolType(req.ToolType)
		filters.ToolType = &toolType
	}

	if req.Scope != "" {
		scope := registry.ToolScope(req.Scope)
		filters.Scope = &scope
	}

	if req.PricingModel != "" {
		pricingModel := registry.PricingModel(req.PricingModel)
		filters.PricingModel = &pricingModel
	}

	if req.Status != "" {
		status := registry.ToolStatus(req.Status)
		filters.Status = &status
	}

	return filters
}
