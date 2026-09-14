package registry

import (
	"errors"

	"github.com/canakyuz/keystone/internal/domain/registry"
	dto "github.com/canakyuz/keystone/internal/dto/registry"
	"github.com/canakyuz/keystone/internal/middleware"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
	"github.com/gofiber/fiber/v2"
)

// ActivationHandler serves the module and tool lifecycle endpoints: install,
// activate, deactivate and uninstall.
//
// Every call passes c.UserContext() down, not c.Context(). The user context is the one
// TenantContextMiddleware filled with the tenant and the acting subject; the fasthttp
// context carries neither.
type ActivationHandler struct {
	activationService *registryService.TenantActivationService
	dependencyChecker *registryService.DependencyCheckerService
}

// NewActivationHandler builds an ActivationHandler.
func NewActivationHandler(
	activationService *registryService.TenantActivationService,
	dependencyChecker *registryService.DependencyCheckerService,
) *ActivationHandler {
	return &ActivationHandler{
		activationService: activationService,
		dependencyChecker: dependencyChecker,
	}
}

// activationOutcomes maps the domain errors a client can act on to their status. Their
// messages are fixed strings written for the client. Anything else is returned to the
// application's error handler, which logs it and answers 500 without its text.
var activationOutcomes = []struct {
	err    error
	status int
}{
	{registry.ErrModuleNotFound, fiber.StatusNotFound},
	{registry.ErrToolNotFound, fiber.StatusNotFound},
	{registry.ErrTenantModuleNotFound, fiber.StatusNotFound},
	{registry.ErrTenantToolNotFound, fiber.StatusNotFound},
	{registry.ErrTenantModuleAlreadyInstalled, fiber.StatusConflict},
	{registry.ErrTenantToolAlreadyInstalled, fiber.StatusConflict},
	{registry.ErrTenantModuleAlreadyActive, fiber.StatusConflict},
	{registry.ErrTenantModuleAlreadyInactive, fiber.StatusConflict},
	{registry.ErrTenantToolAlreadyActive, fiber.StatusConflict},
	{registry.ErrTenantToolAlreadyInactive, fiber.StatusConflict},
	{registry.ErrMissingRequiredDependencies, fiber.StatusConflict},
}

// activationError writes the response for an error from the activation service.
func activationError(c *fiber.Ctx, err error) error {
	for _, outcome := range activationOutcomes {
		if errors.Is(err, outcome.err) {
			return c.Status(outcome.status).JSON(fiber.Map{"error": outcome.err.Error()})
		}
	}
	return err
}

// InstallModule installs a module for the current tenant.
// POST /api/v1/registry/tenant/modules/install
func (h *ActivationHandler) InstallModule(c *fiber.Ctx) error {
	// Take the tenant and user ids set by the middleware.
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Parse the request body into the DTO.
	var req dto.InstallModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Build the request for the service layer.
	serviceReq := registryService.InstallModuleRequest{
		TenantID:        tenantID,
		ModuleID:        req.ModuleID,
		InstalledBy:     userID,
		AutoActivate:    req.AutoActivate,
		AutoInstallDeps: req.AutoInstallDeps,
	}

	// Install the module through the activation service.
	result, err := h.activationService.InstallModule(c.UserContext(), serviceReq)
	if err != nil {
		return activationError(c, err)
	}

	// Turn the service result into an HTTP response.
	response := dto.InstallModuleResponse{
		TenantModule:         dto.ToTenantModuleResponse(result.TenantModule),
		DependencyCheck:      dto.ToDependencyCheckDTO(result.DependencyCheckResult),
		AutoInstalledModules: result.AutoInstalledModules,
		AutoInstalledTools:   result.AutoInstalledTools,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": response})
}

// ActivateModule activates an installed module.
// POST /api/v1/registry/tenant/modules/:module_id/activate
func (h *ActivationHandler) ActivateModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.ActivateModule(c.UserContext(), tenantID, moduleID, userID); err != nil {
		return activationError(c, err)
	}

	return c.JSON(fiber.Map{"message": "module activated"})
}

// DeactivateModule deactivates an active module.
// POST /api/v1/registry/tenant/modules/:module_id/deactivate
func (h *ActivationHandler) DeactivateModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.DeactivateModule(c.UserContext(), tenantID, moduleID, userID); err != nil {
		return activationError(c, err)
	}

	return c.JSON(fiber.Map{"message": "module deactivated"})
}

// UninstallModule removes a module from the tenant entirely.
// DELETE /api/v1/registry/tenant/modules/:module_id
func (h *ActivationHandler) UninstallModule(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.UninstallModule(c.UserContext(), tenantID, moduleID); err != nil {
		return activationError(c, err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CompleteModuleSetup marks a module's post-install steps as done.
// POST /api/v1/registry/tenant/modules/:module_id/complete-setup
func (h *ActivationHandler) CompleteModuleSetup(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.CompleteModuleSetup(c.UserContext(), tenantID, moduleID); err != nil {
		return activationError(c, err)
	}

	return c.JSON(fiber.Map{"message": "module setup completed"})
}

// InstallTool installs a tool for the current tenant.
// POST /api/v1/registry/tenant/tools/install
func (h *ActivationHandler) InstallTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.InstallToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Build the request for the service layer.
	serviceReq := registryService.InstallToolRequest{
		TenantID:        tenantID,
		ToolID:          req.ToolID,
		ModuleID:        req.ModuleID,
		InstalledBy:     userID,
		AutoActivate:    req.AutoActivate,
		AutoInstallDeps: req.AutoInstallDeps,
	}

	result, err := h.activationService.InstallTool(c.UserContext(), serviceReq)
	if err != nil {
		return activationError(c, err)
	}

	// Turn the service result into an HTTP response.
	response := dto.InstallToolResponse{
		TenantTool:         dto.ToTenantToolResponse(result.TenantTool),
		DependencyCheck:    dto.ToDependencyCheckDTO(result.DependencyCheckResult),
		AutoInstalledTools: result.AutoInstalledTools,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": response})
}

// ActivateTool activates an installed tool.
// POST /api/v1/registry/tenant/tools/:tool_id/activate
func (h *ActivationHandler) ActivateTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.ActivateTool(c.UserContext(), tenantID, toolID, userID); err != nil {
		return activationError(c, err)
	}

	return c.JSON(fiber.Map{"message": "tool activated"})
}

// DeactivateTool deactivates an active tool.
// POST /api/v1/registry/tenant/tools/:tool_id/deactivate
func (h *ActivationHandler) DeactivateTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.DeactivateTool(c.UserContext(), tenantID, toolID, userID); err != nil {
		return activationError(c, err)
	}

	return c.JSON(fiber.Map{"message": "tool deactivated"})
}

// UninstallTool removes a tool from the tenant entirely.
// DELETE /api/v1/registry/tenant/tools/:tool_id
func (h *ActivationHandler) UninstallTool(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.UninstallTool(c.UserContext(), tenantID, toolID); err != nil {
		return activationError(c, err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CompleteToolSetup marks a tool's post-install steps as done.
// POST /api/v1/registry/tenant/tools/:tool_id/complete-setup
func (h *ActivationHandler) CompleteToolSetup(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.activationService.CompleteToolSetup(c.UserContext(), tenantID, toolID); err != nil {
		return activationError(c, err)
	}

	return c.JSON(fiber.Map{"message": "tool setup completed"})
}

// GetActivatedModules lists the modules active for the current tenant.
// GET /api/v1/registry/tenant/modules
func (h *ActivationHandler) GetActivatedModules(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	modules, err := h.activationService.GetActivatedModules(c.UserContext(), tenantID)
	if err != nil {
		return err
	}

	// Convert the service result into DTOs.
	response := make([]dto.TenantModuleResponse, len(modules))
	for i, module := range modules {
		response[i] = dto.ToTenantModuleResponse(module)
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetActivatedTools lists the tools active for the current tenant.
// GET /api/v1/registry/tenant/tools
func (h *ActivationHandler) GetActivatedTools(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	tools, err := h.activationService.GetActivatedTools(c.UserContext(), tenantID)
	if err != nil {
		return err
	}

	// Convert the service result into DTOs.
	response := make([]dto.TenantToolResponse, len(tools))
	for i, tool := range tools {
		response[i] = dto.ToTenantToolResponse(tool)
	}

	return c.JSON(fiber.Map{"data": response})
}

// CheckModuleDependencies reports which other modules or tools a module requires.
// GET /api/v1/registry/tenant/modules/:module_id/dependencies
func (h *ActivationHandler) CheckModuleDependencies(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	moduleID := c.Params("module_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	result, err := h.dependencyChecker.CheckModuleDependencies(c.UserContext(), tenantID, moduleID)
	if err != nil {
		return activationError(c, err)
	}

	response := dto.ToDependencyCheckDTO(result)

	return c.JSON(fiber.Map{"data": response})
}

// CheckToolDependencies reports which other modules or tools a tool requires.
// GET /api/v1/registry/tenant/tools/:tool_id/dependencies
func (h *ActivationHandler) CheckToolDependencies(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	toolID := c.Params("tool_id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	result, err := h.dependencyChecker.CheckToolDependencies(c.UserContext(), tenantID, toolID)
	if err != nil {
		return activationError(c, err)
	}

	response := dto.ToDependencyCheckDTO(result)

	return c.JSON(fiber.Map{"data": response})
}
