package registry

import (
	"strconv"

	"github.com/canakyuz/keystone/internal/domain/registry"
	dto "github.com/canakyuz/keystone/internal/dto/registry"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
	"github.com/gofiber/fiber/v2"
)

// ToolCatalogHandler, araç kataloğu (pazaryeri) ile ilgili halka açık HTTP isteklerini yönetir.
type ToolCatalogHandler struct {
	catalogService *registryService.ToolCatalogService
}

// NewToolCatalogHandler, yeni bir ToolCatalogHandler örneği oluşturur.
func NewToolCatalogHandler(catalogService *registryService.ToolCatalogService) *ToolCatalogHandler {
	return &ToolCatalogHandler{
		catalogService: catalogService,
	}
}

// ListPublicTools, pazaryerindeki halka açık araçları listeler.
// GET /api/v1/registry/tools
func (h *ToolCatalogHandler) ListPublicTools(c *fiber.Ctx) error {
	var req dto.ToolListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz sorgu parametreleri"})
	}

	// Sayfalama için varsayılan değerleri ayarla.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	// İstekten gelen parametrelere göre filtreleri oluştur.
	filters := buildToolFilters(req)

	tools, err := h.catalogService.ListPublicTools(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Veritabanından gelen araçları HTTP cevap formatına dönüştür.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{
		"data": response,
		"pagination": dto.PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			TotalItems: len(response), // Not: Bu, toplam öğe sayısı değil, mevcut sayfadaki öğe sayısıdır. Gerçek toplam için ayrı bir sorgu gerekir.
			TotalPages: (len(response) + req.PerPage - 1) / req.PerPage,
		},
	})
}

// SearchTools, pazaryerindeki araçlar içinde arama yapar.
// GET /api/v1/registry/tools/search
func (h *ToolCatalogHandler) SearchTools(c *fiber.Ctx) error {
	var req dto.ToolSearchRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz sorgu parametreleri"})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Arama sorgusu zorunludur"})
	}

	// Sayfalama için varsayılan değerleri ayarla.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	// Arama ve filtreleme için filtreleri oluştur.
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

	// Sonuçları HTTP cevap formatına dönüştür.
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

// GetToolByID, belirtilen ID'ye sahip aracı getirir.
// GET /api/v1/registry/tools/:id
func (h *ToolCatalogHandler) GetToolByID(c *fiber.Ctx) error {
	id := c.Params("id")

	tool, err := h.catalogService.GetToolByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Araç bulunamadı"})
	}

	response := dto.ToToolDetailResponse(tool)

	return c.JSON(fiber.Map{"data": response})
}

// GetToolBySlug, belirtilen 'slug' (kısa isme) sahip aracı getirir.
// GET /api/v1/registry/tools/slug/:slug
func (h *ToolCatalogHandler) GetToolBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	tool, err := h.catalogService.GetToolBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Araç bulunamadı"})
	}

	response := dto.ToToolDetailResponse(tool)

	return c.JSON(fiber.Map{"data": response})
}

// GetPopularTools, en popüler araçları listeler.
// GET /api/v1/registry/tools/popular
func (h *ToolCatalogHandler) GetPopularTools(c *fiber.Ctx) error {
	limit := 10 // Varsayılan olarak 10 araç getir.
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit // Sorgu parametresi varsa ve geçerliyse limiti güncelle.
		}
	}

	tools, err := h.catalogService.GetPopularTools(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Sonuçları HTTP cevap formatına dönüştür.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetTopRatedTools, en yüksek puanlı araçları listeler.
// GET /api/v1/registry/tools/top-rated
func (h *ToolCatalogHandler) GetTopRatedTools(c *fiber.Ctx) error {
	limit := 10 // Varsayılan limit.
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	tools, err := h.catalogService.GetTopRatedTools(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Sonuçları HTTP cevap formatına dönüştür.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetToolsByCategory, belirli bir kategorideki araçları listeler.
// GET /api/v1/registry/tools/category/:category
func (h *ToolCatalogHandler) GetToolsByCategory(c *fiber.Ctx) error {
	categoryParam := c.Params("category")
	category := registry.ToolCategory(categoryParam)

	var req dto.ToolListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz sorgu parametreleri"})
	}

	// Sayfalama için varsayılan değerleri ayarla.
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

	// Sonuçları HTTP cevap formatına dönüştür.
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

// GetFreeTools, ücretsiz olan araçları listeler.
// GET /api/v1/registry/tools/free
func (h *ToolCatalogHandler) GetFreeTools(c *fiber.Ctx) error {
	var req dto.ToolListRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz sorgu parametreleri"})
	}

	// Sayfalama için varsayılan değerleri ayarla.
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

	// Sonuçları HTTP cevap formatına dönüştür.
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

// GetNewTools, pazaryerine yeni eklenmiş araçları listeler.
// GET /api/v1/registry/tools/new
func (h *ToolCatalogHandler) GetNewTools(c *fiber.Ctx) error {
	limit := 10 // Varsayılan limit.
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	tools, err := h.catalogService.GetNewTools(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Sonuçları HTTP cevap formatına dönüştür.
	response := make([]dto.ToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// buildToolFilters, HTTP isteğindeki sorgu parametrelerinden (query params) veritabanı sorgusu için bir filtre nesnesi oluşturur.
// Bu yardımcı fonksiyon, kod tekrarını önler ve filtreleme mantığını merkezileştirir.
func buildToolFilters(req dto.ToolListRequest) registry.ToolFilters {
	filters := registry.ToolFilters{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.PerPage,
		Offset:    (req.Page - 1) * req.PerPage,
		IsPublic:  req.IsPublic,
		IsBeta:    req.IsBeta,
	}

	// String gelen filtre parametrelerini, domain katmanındaki özel tiplere dönüştür.
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
