package registry

import (
	dto "github.com/canakyuz/keystone/internal/dto/registry"
	"github.com/canakyuz/keystone/internal/middleware"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
	"github.com/gofiber/fiber/v2"
)

// ActivationHandler, modül ve araçların aktivasyon/deaktivasyon gibi yaşam döngüsüyle ilgili HTTP isteklerini yönetir.
type ActivationHandler struct {
	activationService *registryService.TenantActivationService
	dependencyChecker *registryService.DependencyCheckerService
}

// NewActivationHandler, yeni bir ActivationHandler örneği oluşturur.
// Bu yapıcı metod, bağımlılıkların enjekte edilmesini (dependency injection) sağlar.
func NewActivationHandler(
	activationService *registryService.TenantActivationService,
	dependencyChecker *registryService.DependencyCheckerService,
) *ActivationHandler {
	return &ActivationHandler{
		activationService: activationService,
		dependencyChecker: dependencyChecker,
	}
}

// InstallModule, mevcut kiracı (tenant) için bir modül kurar.
// POST /api/v1/registry/tenant/modules/install
func (h *ActivationHandler) InstallModule(c *fiber.Ctx) error {
	// Middleware'den kiracı ve kullanıcı ID'lerini al.
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	// Gelen isteğin gövdesini (body) DTO'ya (Data Transfer Object) parse et.
	var req dto.InstallModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi"})
	}

	// Servis katmanına gönderilecek isteği oluştur.
	serviceReq := registryService.InstallModuleRequest{
		TenantID:        tenantID,
		ModuleID:        req.ModuleID,
		InstalledBy:     userID,
		AutoActivate:    req.AutoActivate,
		AutoInstallDeps: req.AutoInstallDeps,
	}

	// Aktivasyon servisini çağırarak modülü kur.
	result, err := h.activationService.InstallModule(c.Context(), serviceReq)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Servisten gelen sonucu HTTP cevabına dönüştür.
	response := dto.InstallModuleResponse{
		TenantModule:         dto.ToTenantModuleResponse(result.TenantModule),
		DependencyCheck:      dto.ToDependencyCheckDTO(result.DependencyCheckResult),
		AutoInstalledModules: result.AutoInstalledModules,
		AutoInstalledTools:   result.AutoInstalledTools,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": response})
}

// ActivateModule, kurulmuş bir modülü aktif hale getirir.
// POST /api/v1/registry/tenant/modules/:module_id/activate
func (h *ActivationHandler) ActivateModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.ActivateModule(c.Context(), tenantID, moduleID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Modül başarıyla aktive edildi"})
}

// DeactivateModule, aktif bir modülü pasif hale getirir.
// POST /api/v1/registry/tenant/modules/:module_id/deactivate
func (h *ActivationHandler) DeactivateModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.DeactivateModule(c.Context(), tenantID, moduleID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Modül başarıyla devre dışı bırakıldı"})
}

// UninstallModule, bir modülü kiracıdan tamamen kaldırır.
// DELETE /api/v1/registry/tenant/modules/:module_id
func (h *ActivationHandler) UninstallModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.UninstallModule(c.Context(), tenantID, moduleID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CompleteModuleSetup, bir modülün kurulum sonrası adımlarının tamamlandığını işaretler.
// POST /api/v1/registry/tenant/modules/:module_id/complete-setup
func (h *ActivationHandler) CompleteModuleSetup(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.CompleteModuleSetup(c.Context(), tenantID, moduleID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Modül kurulumu başarıyla tamamlandı"})
}

// InstallTool, mevcut kiracı için bir araç kurar.
// POST /api/v1/registry/tenant/tools/install
func (h *ActivationHandler) InstallTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	var req dto.InstallToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi"})
	}

	// Servis katmanına gönderilecek isteği oluştur.
	serviceReq := registryService.InstallToolRequest{
		TenantID:        tenantID,
		ToolID:          req.ToolID,
		ModuleID:        req.ModuleID,
		InstalledBy:     userID,
		AutoActivate:    req.AutoActivate,
		AutoInstallDeps: req.AutoInstallDeps,
	}

	result, err := h.activationService.InstallTool(c.Context(), serviceReq)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Servisten gelen sonucu HTTP cevabına dönüştür.
	response := dto.InstallToolResponse{
		TenantTool:         dto.ToTenantToolResponse(result.TenantTool),
		DependencyCheck:    dto.ToDependencyCheckDTO(result.DependencyCheckResult),
		AutoInstalledTools: result.AutoInstalledTools,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": response})
}

// ActivateTool, kurulmuş bir aracı aktif hale getirir.
// POST /api/v1/registry/tenant/tools/:tool_id/activate
func (h *ActivationHandler) ActivateTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.ActivateTool(c.Context(), tenantID, toolID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Araç başarıyla aktive edildi"})
}

// DeactivateTool, aktif bir aracı pasif hale getirir.
// POST /api/v1/registry/tenant/tools/:tool_id/deactivate
func (h *ActivationHandler) DeactivateTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.DeactivateTool(c.Context(), tenantID, toolID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Araç başarıyla devre dışı bırakıldı"})
}

// UninstallTool, bir aracı kiracıdan tamamen kaldırır.
// DELETE /api/v1/registry/tenant/tools/:tool_id
func (h *ActivationHandler) UninstallTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.UninstallTool(c.Context(), tenantID, toolID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CompleteToolSetup, bir aracın kurulum sonrası adımlarının tamamlandığını işaretler.
// POST /api/v1/registry/tenant/tools/:tool_id/complete-setup
func (h *ActivationHandler) CompleteToolSetup(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	if err := h.activationService.CompleteToolSetup(c.Context(), tenantID, toolID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Araç kurulumu başarıyla tamamlandı"})
}

// GetActivatedModules, mevcut kiracı için aktif olan modülleri listeler.
// GET /api/v1/registry/tenant/modules
func (h *ActivationHandler) GetActivatedModules(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	modules, err := h.activationService.GetActivatedModules(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Servisten gelen veriyi DTO'ya çevir.
	response := make([]dto.TenantModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToTenantModuleResponse(module)
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetActivatedTools, mevcut kiracı için aktif olan araçları listeler.
// GET /api/v1/registry/tenant/tools
func (h *ActivationHandler) GetActivatedTools(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	tools, err := h.activationService.GetActivatedTools(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Servisten gelen veriyi DTO'ya çevir.
	response := make([]dto.TenantToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToTenantToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// CheckModuleDependencies, bir modülün kurulum için gerekli olan diğer modül/araç bağımlılıklarını kontrol eder.
// GET /api/v1/registry/tenant/modules/:module_id/dependencies
func (h *ActivationHandler) CheckModuleDependencies(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	result, err := h.dependencyChecker.CheckModuleDependencies(c.Context(), tenantID, moduleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	response := dto.ToDependencyCheckDTO(result)

	return c.JSON(fiber.Map{"data": response})
}

// CheckToolDependencies, bir aracın kurulum için gerekli olan diğer modül/araç bağımlılıklarını kontrol eder.
// GET /api/v1/registry/tenant/tools/:tool_id/dependencies
func (h *ActivationHandler) CheckToolDependencies(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkisiz erişim"})
	}

	result, err := h.dependencyChecker.CheckToolDependencies(c.Context(), tenantID, toolID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	response := dto.ToDependencyCheckDTO(result)

	return c.JSON(fiber.Map{"data": response})
}
