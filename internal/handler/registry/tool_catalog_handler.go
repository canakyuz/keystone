package registry

import (
	"strconv"

	"github.com/canakyuz/keystone/internal/domain/registry"
	dto "github.com/canakyuz/keystone/internal/dto/registry"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
	"github.com/gofiber/fiber/v2"
)

// ToolCatalogHandler serves the public tool catalogue endpoints.
type ToolCatalogHandler struct {
	catalogService *registryService.ToolCatalogService
}

// NewToolCatalogHandler builds a ToolCatalogHandler.
func NewToolCatalogHandler(catalogService *registryService.ToolCatalogService) *ToolCatalogHandler {
	return &ToolCatalogHandler{
		catalogService: catalogService,
	}
}

// ListPublicTools lists the publicly visible tools in the marketplace.
// GET /api/v1/registry/tools
func (h *ToolCatalogHandler) ListPublicTools(c *fiber.Ctx) error {
	var req dto.ToolListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query parameters"})
	}

	// Apply the pagination defaults.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	// Build the filters from the query parameters.
	filters := buildToolFilters(req)

	tools, err := h.catalogService.ListPublicTools(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the tools into the HTTP response shape.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"pagination": dto.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			TotalItems: len(response), // Note: this is the size of the current page, not the true
			// total. A separate count query would be needed for that.
			TotalPages: (len(response) + req.PerPage - 1) / req.PerPage,
		},
	})
}

// SearchTools searches the tools in the marketplace.
// GET /api/v1/registry/tools/search
func (h *ToolCatalogHandler) SearchTools(c *fiber.Ctx) error {
	var req dto.ToolSearchRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query parameters"})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "a search query is required"})
	}

	// Apply the pagination defaults.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	// Build the search and filter criteria.
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the results into the HTTP response shape.
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

// GetToolByID returns the tool with the given id.
// GET /api/v1/registry/tools/:id
func (h *ToolCatalogHandler) GetToolByID(c *fiber.Ctx) error {
	id := c.Params("id")

	tool, err := h.catalogService.GetToolByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tool not found"})
	}

	response := dto.ToToolDetailResponse(tool)

	return c.JSON(fiber.Map{"data": response})
}

// GetToolBySlug returns the tool with the given slug.
// GET /api/v1/registry/tools/slug/:slug
func (h *ToolCatalogHandler) GetToolBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	tool, err := h.catalogService.GetToolBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tool not found"})
	}

	response := dto.ToToolDetailResponse(tool)

	return c.JSON(fiber.Map{"data": response})
}

// GetPopularTools lists the most popular tools.
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the results into the HTTP response shape.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetTopRatedTools lists the highest rated tools.
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the results into the HTTP response shape.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetToolsByCategory lists the tools in one category.
// GET /api/v1/registry/tools/category/:category
func (h *ToolCatalogHandler) GetToolsByCategory(c *fiber.Ctx) error {
	categoryParam := c.Params("category")
	category := registry.ToolCategory(categoryParam)

	var req dto.ToolListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query parameters"})
	}

	// Apply the pagination defaults.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	filters := buildToolFilters(req)

	tools, err := h.catalogService.GetToolsByCategory(c.Context(), category, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the results into the HTTP response shape.
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

// GetFreeTools lists the free tools.
// GET /api/v1/registry/tools/free
func (h *ToolCatalogHandler) GetFreeTools(c *fiber.Ctx) error {
	var req dto.ToolListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query parameters"})
	}

	// Apply the pagination defaults.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	filters := buildToolFilters(req)

	tools, err := h.catalogService.GetFreeTools(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the results into the HTTP response shape.
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

// GetNewTools lists the tools most recently added to the marketplace.
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert the results into the HTTP response shape.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// buildToolFilters turns the request's query parameters into a filter object for the
// database query, keeping the filtering logic in one place.
func buildToolFilters(req dto.ToolListRequest) registry.ToolFilters {
	filters := registry.ToolFilters{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PerPage,
		Offset:    (req.Page - 1) * req.PerPage,
		IsPublic:  req.IsPublic,
		IsBeta:    req.IsBeta,
	}

	// Convert the string parameters into the domain layer's own types.
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
