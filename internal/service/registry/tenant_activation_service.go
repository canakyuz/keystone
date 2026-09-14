package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// TenantActivationService handles module and tool activation/deactivation
//
// The install counts on modules and tools are not touched here. A trigger on the
// installation tables keeps them, counting active installations; bumping them from the
// service as well counted every installation twice.
type TenantActivationService struct {
	moduleRepo        registry.ModuleRepository
	toolRepo          registry.ToolRepository
	tenantModuleRepo  registry.TenantModuleRepository
	tenantToolRepo    registry.TenantToolRepository
	dependencyChecker *DependencyCheckerService
}

// NewTenantActivationService creates a new tenant activation service
func NewTenantActivationService(
	moduleRepo registry.ModuleRepository,
	toolRepo registry.ToolRepository,
	tenantModuleRepo registry.TenantModuleRepository,
	tenantToolRepo registry.TenantToolRepository,
	dependencyChecker *DependencyCheckerService,
) *TenantActivationService {
	return &TenantActivationService{
		moduleRepo:        moduleRepo,
		toolRepo:          toolRepo,
		tenantModuleRepo:  tenantModuleRepo,
		tenantToolRepo:    tenantToolRepo,
		dependencyChecker: dependencyChecker,
	}
}

// InstallModuleRequest represents module installation request
type InstallModuleRequest struct {
	TenantID        string
	ModuleID        string
	InstalledBy     string
	AutoActivate    bool
	AutoInstallDeps bool
}

// InstallModuleResponse represents module installation response
type InstallModuleResponse struct {
	TenantModule          *registry.TenantModule
	DependencyCheckResult *DependencyCheckResult
	AutoInstalledModules  []string
	AutoInstalledTools    []string
}

// InstallModule installs a module for a tenant
func (s *TenantActivationService) InstallModule(ctx context.Context, req InstallModuleRequest) (*InstallModuleResponse, error) {
	// Validate module exists
	module, err := s.moduleRepo.GetByID(ctx, req.ModuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up the module: %w", err)
	}

	// Refuse early, before any dependency is installed on this installation's behalf. The
	// insert refuses a duplicate on its own as well, so two concurrent requests cannot both
	// get through.
	if _, err := s.tenantModuleRepo.GetByTenantAndModule(ctx, req.TenantID, req.ModuleID); err == nil {
		return nil, registry.ErrTenantModuleAlreadyInstalled
	} else if !errors.Is(err, registry.ErrTenantModuleNotFound) {
		return nil, fmt.Errorf("failed to check the installation: %w", err)
	}

	// Check dependencies
	depResult, err := s.dependencyChecker.CheckModuleDependencies(ctx, req.TenantID, req.ModuleID)
	if err != nil {
		return nil, fmt.Errorf("dependency check failed: %w", err)
	}

	response := &InstallModuleResponse{
		DependencyCheckResult: depResult,
		AutoInstalledModules:  []string{},
		AutoInstalledTools:    []string{},
	}

	// Auto-install dependencies if requested
	if req.AutoInstallDeps {
		for _, dep := range depResult.AutoInstallSuggestions {
			if dep.Type == "module" {
				autoReq := InstallModuleRequest{
					TenantID:        req.TenantID,
					ModuleID:        dep.ID,
					InstalledBy:     req.InstalledBy,
					AutoActivate:    true,
					AutoInstallDeps: true,
				}
				_, err := s.InstallModule(ctx, autoReq)
				if err != nil {
					return nil, fmt.Errorf("failed to auto-install module %s: %w", dep.Code, err)
				}
				response.AutoInstalledModules = append(response.AutoInstalledModules, dep.Code)
			} else if dep.Type == "tool" {
				autoReq := InstallToolRequest{
					TenantID:        req.TenantID,
					ToolID:          dep.ID,
					InstalledBy:     req.InstalledBy,
					AutoActivate:    true,
					AutoInstallDeps: true,
				}
				_, err := s.InstallTool(ctx, autoReq)
				if err != nil {
					return nil, fmt.Errorf("failed to auto-install tool %s: %w", dep.Code, err)
				}
				response.AutoInstalledTools = append(response.AutoInstalledTools, dep.Code)
			}
		}

		// Re-check dependencies after auto-install
		depResult, err = s.dependencyChecker.CheckModuleDependencies(ctx, req.TenantID, req.ModuleID)
		if err != nil {
			return nil, fmt.Errorf("dependency re-check failed: %w", err)
		}
		response.DependencyCheckResult = depResult
	}

	// Check if all required dependencies are met
	if !depResult.CanActivate && req.AutoActivate {
		return nil, fmt.Errorf("cannot activate module: %w", registry.ErrMissingRequiredDependencies)
	}

	// Create tenant module record
	tenantModule, err := registry.NewTenantModule(req.TenantID, req.ModuleID, module.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant module: %w", err)
	}

	tenantModule.CreatedBy = req.InstalledBy

	// Create in database
	if err := s.tenantModuleRepo.Create(ctx, tenantModule); err != nil {
		return nil, fmt.Errorf("failed to save tenant module: %w", err)
	}

	// Auto-activate if requested and dependencies are met
	if req.AutoActivate && depResult.CanActivate {
		if err := s.tenantModuleRepo.Activate(ctx, req.TenantID, req.ModuleID, req.InstalledBy); err != nil {
			return nil, fmt.Errorf("failed to activate module: %w", err)
		}

		// Reload to get updated data
		tenantModule, err = s.tenantModuleRepo.GetByTenantAndModule(ctx, req.TenantID, req.ModuleID)
		if err != nil {
			return nil, fmt.Errorf("failed to reload tenant module: %w", err)
		}
	}

	response.TenantModule = tenantModule
	return response, nil
}

// ActivateModule activates an installed module
func (s *TenantActivationService) ActivateModule(ctx context.Context, tenantID, moduleID, activatedBy string) error {
	// A module that is not installed is not found, whatever its dependencies say.
	if _, err := s.tenantModuleRepo.GetByTenantAndModule(ctx, tenantID, moduleID); err != nil {
		return fmt.Errorf("failed to look up the installation: %w", err)
	}

	// Check dependencies
	depResult, err := s.dependencyChecker.CheckModuleDependencies(ctx, tenantID, moduleID)
	if err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	if !depResult.CanActivate {
		return fmt.Errorf("cannot activate module: %w", registry.ErrMissingRequiredDependencies)
	}

	// Activate
	if err := s.tenantModuleRepo.Activate(ctx, tenantID, moduleID, activatedBy); err != nil {
		return fmt.Errorf("failed to activate module: %w", err)
	}

	return nil
}

// DeactivateModule deactivates a module
func (s *TenantActivationService) DeactivateModule(ctx context.Context, tenantID, moduleID, deactivatedBy string) error {
	if err := s.tenantModuleRepo.Deactivate(ctx, tenantID, moduleID, deactivatedBy); err != nil {
		return fmt.Errorf("failed to deactivate module: %w", err)
	}

	return nil
}

// UninstallModule soft deletes a tenant module
func (s *TenantActivationService) UninstallModule(ctx context.Context, tenantID, moduleID string) error {
	if err := s.tenantModuleRepo.Uninstall(ctx, tenantID, moduleID); err != nil {
		return fmt.Errorf("failed to uninstall module: %w", err)
	}

	return nil
}

// CompleteModuleSetup marks module setup as completed
func (s *TenantActivationService) CompleteModuleSetup(ctx context.Context, tenantID, moduleID string) error {
	tenantModule, err := s.tenantModuleRepo.GetByTenantAndModule(ctx, tenantID, moduleID)
	if err != nil {
		return fmt.Errorf("failed to look up the installation: %w", err)
	}

	tenantModule.CompleteSetup()

	if err := s.tenantModuleRepo.Update(ctx, tenantModule); err != nil {
		return fmt.Errorf("failed to complete module setup: %w", err)
	}

	return nil
}

// InstallToolRequest represents tool installation request
type InstallToolRequest struct {
	TenantID        string
	ToolID          string
	ModuleID        string // Optional: if tool is installed via module
	InstalledBy     string
	AutoActivate    bool
	AutoInstallDeps bool
}

// InstallToolResponse represents tool installation response
type InstallToolResponse struct {
	TenantTool            *registry.TenantTool
	DependencyCheckResult *DependencyCheckResult
	AutoInstalledTools    []string
}

// InstallTool installs a tool for a tenant
func (s *TenantActivationService) InstallTool(ctx context.Context, req InstallToolRequest) (*InstallToolResponse, error) {
	// Validate tool exists
	tool, err := s.toolRepo.GetByID(ctx, req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up the tool: %w", err)
	}

	// The module a tool is installed through is optional, but one that is named must exist;
	// the column references modules.
	if req.ModuleID != "" {
		if _, err := s.moduleRepo.GetByID(ctx, req.ModuleID); err != nil {
			return nil, fmt.Errorf("failed to look up the module: %w", err)
		}
	}

	// Refuse early, on the rules of InstallModule.
	if _, err := s.tenantToolRepo.GetByTenantAndTool(ctx, req.TenantID, req.ToolID); err == nil {
		return nil, registry.ErrTenantToolAlreadyInstalled
	} else if !errors.Is(err, registry.ErrTenantToolNotFound) {
		return nil, fmt.Errorf("failed to check the installation: %w", err)
	}

	// Check dependencies
	depResult, err := s.dependencyChecker.CheckToolDependencies(ctx, req.TenantID, req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("dependency check failed: %w", err)
	}

	response := &InstallToolResponse{
		DependencyCheckResult: depResult,
		AutoInstalledTools:    []string{},
	}

	// Auto-install dependencies if requested
	if req.AutoInstallDeps {
		for _, dep := range depResult.AutoInstallSuggestions {
			if dep.Type == "tool" {
				autoReq := InstallToolRequest{
					TenantID:        req.TenantID,
					ToolID:          dep.ID,
					InstalledBy:     req.InstalledBy,
					AutoActivate:    true,
					AutoInstallDeps: true,
				}
				_, err := s.InstallTool(ctx, autoReq)
				if err != nil {
					return nil, fmt.Errorf("failed to auto-install tool %s: %w", dep.Code, err)
				}
				response.AutoInstalledTools = append(response.AutoInstalledTools, dep.Code)
			}
		}

		// Re-check dependencies after auto-install
		depResult, err = s.dependencyChecker.CheckToolDependencies(ctx, req.TenantID, req.ToolID)
		if err != nil {
			return nil, fmt.Errorf("dependency re-check failed: %w", err)
		}
		response.DependencyCheckResult = depResult
	}

	// Check if all required dependencies are met
	if !depResult.CanActivate && req.AutoActivate {
		return nil, fmt.Errorf("cannot activate tool: %w", registry.ErrMissingRequiredDependencies)
	}

	// Create tenant tool record
	tenantTool, err := registry.NewTenantTool(req.TenantID, req.ToolID, tool.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant tool: %w", err)
	}

	tenantTool.ModuleID = req.ModuleID
	tenantTool.CreatedBy = req.InstalledBy

	// Create in database
	if err := s.tenantToolRepo.Create(ctx, tenantTool); err != nil {
		return nil, fmt.Errorf("failed to save tenant tool: %w", err)
	}

	// Auto-activate if requested and dependencies are met
	if req.AutoActivate && depResult.CanActivate {
		if err := s.tenantToolRepo.Activate(ctx, req.TenantID, req.ToolID, req.InstalledBy); err != nil {
			return nil, fmt.Errorf("failed to activate tool: %w", err)
		}

		// Reload to get updated data
		tenantTool, err = s.tenantToolRepo.GetByTenantAndTool(ctx, req.TenantID, req.ToolID)
		if err != nil {
			return nil, fmt.Errorf("failed to reload tenant tool: %w", err)
		}
	}

	response.TenantTool = tenantTool
	return response, nil
}

// ActivateTool activates an installed tool
func (s *TenantActivationService) ActivateTool(ctx context.Context, tenantID, toolID, activatedBy string) error {
	// A tool that is not installed is not found, whatever its dependencies say.
	if _, err := s.tenantToolRepo.GetByTenantAndTool(ctx, tenantID, toolID); err != nil {
		return fmt.Errorf("failed to look up the installation: %w", err)
	}

	// Check dependencies
	depResult, err := s.dependencyChecker.CheckToolDependencies(ctx, tenantID, toolID)
	if err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	if !depResult.CanActivate {
		return fmt.Errorf("cannot activate tool: %w", registry.ErrMissingRequiredDependencies)
	}

	// Activate
	if err := s.tenantToolRepo.Activate(ctx, tenantID, toolID, activatedBy); err != nil {
		return fmt.Errorf("failed to activate tool: %w", err)
	}

	return nil
}

// DeactivateTool deactivates a tool
func (s *TenantActivationService) DeactivateTool(ctx context.Context, tenantID, toolID, deactivatedBy string) error {
	if err := s.tenantToolRepo.Deactivate(ctx, tenantID, toolID, deactivatedBy); err != nil {
		return fmt.Errorf("failed to deactivate tool: %w", err)
	}

	return nil
}

// UninstallTool soft deletes a tenant tool
func (s *TenantActivationService) UninstallTool(ctx context.Context, tenantID, toolID string) error {
	if err := s.tenantToolRepo.Uninstall(ctx, tenantID, toolID); err != nil {
		return fmt.Errorf("failed to uninstall tool: %w", err)
	}

	return nil
}

// CompleteToolSetup marks tool setup as completed
func (s *TenantActivationService) CompleteToolSetup(ctx context.Context, tenantID, toolID string) error {
	tenantTool, err := s.tenantToolRepo.GetByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return fmt.Errorf("failed to look up the installation: %w", err)
	}

	tenantTool.CompleteSetup()

	if err := s.tenantToolRepo.Update(ctx, tenantTool); err != nil {
		return fmt.Errorf("failed to complete tool setup: %w", err)
	}

	return nil
}

// GetActivatedModules retrieves all activated modules for a tenant
func (s *TenantActivationService) GetActivatedModules(ctx context.Context, tenantID string) ([]*registry.TenantModule, error) {
	modules, err := s.tenantModuleRepo.ListActivatedByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activated modules: %w", err)
	}

	return modules, nil
}

// GetActivatedTools retrieves all activated tools for a tenant
func (s *TenantActivationService) GetActivatedTools(ctx context.Context, tenantID string) ([]*registry.TenantTool, error) {
	tools, err := s.tenantToolRepo.ListActivatedByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activated tools: %w", err)
	}

	return tools, nil
}
