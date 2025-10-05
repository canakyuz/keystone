package registry

import (
	"github.com/gofiber/fiber/v2"
	"nexpaces-api/internal/middleware"
	dto "nexpaces-api/internal/dto/registry"
	registryService "nexpaces-api/internal/service/registry"
)

// ActivationHandler handles module/tool activation HTTP requests
type ActivationHandler struct {
	activationService *registryService.TenantActivationService
	dependencyChecker *registryService.DependencyCheckerService
}

// NewActivationHandler creates a new activation handler
func NewActivationHandler(
	activationService *registryService.TenantActivationService,
	dependencyChecker *registryService.DependencyCheckerService,
) *ActivationHandler {
	return &ActivationHandler{
		activationService: activationService,
		dependencyChecker: dependencyChecker,
	}
}

// InstallModule installs a module for the current tenant
// POST /api/v1/registry/tenant/modules/install
func (h *ActivationHandler) InstallModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req dto.InstallModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build service request
	serviceReq := registryService.InstallModuleRequest{
		TenantID:        tenantID,
		ModuleID:        req.ModuleID,
		InstalledBy:     userID,
		AutoActivate:    req.AutoActivate,
		AutoInstallDeps: req.AutoInstallDeps,
	}

	result, err := h.activationService.InstallModule(c.Context(), serviceReq)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := dto.InstallModuleResponse{
		TenantModule:         dto.ToTenantModuleResponse(result.TenantModule),
		DependencyCheck:      dto.ToDependencyCheckDTO(result.DependencyCheckResult),
		AutoInstalledModules: result.AutoInstalledModules,
		AutoInstalledTools:   result.AutoInstalledTools,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": response,
	})
}

// ActivateModule activates an installed module
// POST /api/v1/registry/tenant/modules/:module_id/activate
func (h *ActivationHandler) ActivateModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.ActivateModule(c.Context(), tenantID, moduleID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Module activated successfully",
	})
}

// DeactivateModule deactivates a module
// POST /api/v1/registry/tenant/modules/:module_id/deactivate
func (h *ActivationHandler) DeactivateModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.DeactivateModule(c.Context(), tenantID, moduleID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Module deactivated successfully",
	})
}

// UninstallModule uninstalls a module
// DELETE /api/v1/registry/tenant/modules/:module_id
func (h *ActivationHandler) UninstallModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.UninstallModule(c.Context(), tenantID, moduleID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CompleteModuleSetup marks module setup as completed
// POST /api/v1/registry/tenant/modules/:module_id/complete-setup
func (h *ActivationHandler) CompleteModuleSetup(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.CompleteModuleSetup(c.Context(), tenantID, moduleID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Module setup completed successfully",
	})
}

// InstallTool installs a tool for the current tenant
// POST /api/v1/registry/tenant/tools/install
func (h *ActivationHandler) InstallTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req dto.InstallToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build service request
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := dto.InstallToolResponse{
		TenantTool:         dto.ToTenantToolResponse(result.TenantTool),
		DependencyCheck:    dto.ToDependencyCheckDTO(result.DependencyCheckResult),
		AutoInstalledTools: result.AutoInstalledTools,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": response,
	})
}

// ActivateTool activates an installed tool
// POST /api/v1/registry/tenant/tools/:tool_id/activate
func (h *ActivationHandler) ActivateTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.ActivateTool(c.Context(), tenantID, toolID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tool activated successfully",
	})
}

// DeactivateTool deactivates a tool
// POST /api/v1/registry/tenant/tools/:tool_id/deactivate
func (h *ActivationHandler) DeactivateTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.DeactivateTool(c.Context(), tenantID, toolID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tool deactivated successfully",
	})
}

// UninstallTool uninstalls a tool
// DELETE /api/v1/registry/tenant/tools/:tool_id
func (h *ActivationHandler) UninstallTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.UninstallTool(c.Context(), tenantID, toolID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CompleteToolSetup marks tool setup as completed
// POST /api/v1/registry/tenant/tools/:tool_id/complete-setup
func (h *ActivationHandler) CompleteToolSetup(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.activationService.CompleteToolSetup(c.Context(), tenantID, toolID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tool setup completed successfully",
	})
}

// GetActivatedModules retrieves activated modules for current tenant
// GET /api/v1/registry/tenant/modules
func (h *ActivationHandler) GetActivatedModules(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	modules, err := h.activationService.GetActivatedModules(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.TenantModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToTenantModuleResponse(module)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// GetActivatedTools retrieves activated tools for current tenant
// GET /api/v1/registry/tenant/tools
func (h *ActivationHandler) GetActivatedTools(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	tools, err := h.activationService.GetActivatedTools(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response
	response := make([]dto.TenantToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToTenantToolResponse(tool)
	}

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// CheckModuleDependencies checks dependencies for a module
// GET /api/v1/registry/tenant/modules/:module_id/dependencies
func (h *ActivationHandler) CheckModuleDependencies(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	result, err := h.dependencyChecker.CheckModuleDependencies(c.Context(), tenantID, moduleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := dto.ToDependencyCheckDTO(result)

	return c.JSON(fiber.Map{
		"data": response,
	})
}

// CheckToolDependencies checks dependencies for a tool
// GET /api/v1/registry/tenant/tools/:tool_id/dependencies
func (h *ActivationHandler) CheckToolDependencies(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	result, err := h.dependencyChecker.CheckToolDependencies(c.Context(), tenantID, toolID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := dto.ToDependencyCheckDTO(result)

	return c.JSON(fiber.Map{
		"data": response,
	})
}
