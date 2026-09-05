package registry

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// DependencyInfo represents dependency information
type DependencyInfo struct {
	Type                    string `json:"type"` // "module" or "tool"
	ID                      string `json:"id"`
	Code                    string `json:"code"`
	Name                    string `json:"name"`
	DependencyType          string `json:"dependency_type"` // "required", "optional", "recommended"
	AutoInstallOnActivation bool   `json:"auto_install_on_activation"`
	InstallOrder            int    `json:"install_order"`
	AlreadyActivated        bool   `json:"already_activated"`
}

// DependencyCheckResult represents the result of dependency validation
type DependencyCheckResult struct {
	CanActivate            bool             `json:"can_activate"`
	MissingRequiredModules []DependencyInfo `json:"missing_required_modules"`
	MissingRequiredTools   []DependencyInfo `json:"missing_required_tools"`
	RecommendedModules     []DependencyInfo `json:"recommended_modules"`
	RecommendedTools       []DependencyInfo `json:"recommended_tools"`
	OptionalModules        []DependencyInfo `json:"optional_modules"`
	OptionalTools          []DependencyInfo `json:"optional_tools"`
	AutoInstallSuggestions []DependencyInfo `json:"auto_install_suggestions"`
}

// DependencyCheckerService handles dependency validation and resolution
type DependencyCheckerService struct {
	db               *sql.DB
	moduleRepo       registry.ModuleRepository
	toolRepo         registry.ToolRepository
	tenantModuleRepo registry.TenantModuleRepository
	tenantToolRepo   registry.TenantToolRepository
}

// NewDependencyCheckerService creates a new dependency checker service
func NewDependencyCheckerService(
	db *sql.DB,
	moduleRepo registry.ModuleRepository,
	toolRepo registry.ToolRepository,
	tenantModuleRepo registry.TenantModuleRepository,
	tenantToolRepo registry.TenantToolRepository,
) *DependencyCheckerService {
	return &DependencyCheckerService{
		db:               db,
		moduleRepo:       moduleRepo,
		toolRepo:         toolRepo,
		tenantModuleRepo: tenantModuleRepo,
		tenantToolRepo:   tenantToolRepo,
	}
}

// CheckModuleDependencies validates all dependencies for a module activation
func (s *DependencyCheckerService) CheckModuleDependencies(ctx context.Context, tenantID, moduleID string) (*DependencyCheckResult, error) {
	result := &DependencyCheckResult{
		CanActivate:            true,
		MissingRequiredModules: []DependencyInfo{},
		MissingRequiredTools:   []DependencyInfo{},
		RecommendedModules:     []DependencyInfo{},
		RecommendedTools:       []DependencyInfo{},
		OptionalModules:        []DependencyInfo{},
		OptionalTools:          []DependencyInfo{},
		AutoInstallSuggestions: []DependencyInfo{},
	}

	// Get module dependencies
	moduleDeps, err := s.getModuleDependencies(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get module dependencies: %w", err)
	}

	// Check each module dependency
	for _, dep := range moduleDeps {
		isActivated, err := s.tenantModuleRepo.IsModuleActivated(ctx, tenantID, dep.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check module activation: %w", err)
		}

		dep.AlreadyActivated = isActivated

		// Categorize based on dependency type and activation status
		switch dep.DependencyType {
		case "required":
			if !isActivated {
				result.MissingRequiredModules = append(result.MissingRequiredModules, dep)
				result.CanActivate = false
			}
			if dep.AutoInstallOnActivation && !isActivated {
				result.AutoInstallSuggestions = append(result.AutoInstallSuggestions, dep)
			}
		case "recommended":
			if !isActivated {
				result.RecommendedModules = append(result.RecommendedModules, dep)
			}
			if dep.AutoInstallOnActivation && !isActivated {
				result.AutoInstallSuggestions = append(result.AutoInstallSuggestions, dep)
			}
		case "optional":
			if !isActivated {
				result.OptionalModules = append(result.OptionalModules, dep)
			}
		}
	}

	// Get tool dependencies
	toolDeps, err := s.getModuleToolDependencies(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool dependencies: %w", err)
	}

	// Check each tool dependency
	for _, dep := range toolDeps {
		isActivated, err := s.tenantToolRepo.IsToolActivated(ctx, tenantID, dep.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check tool activation: %w", err)
		}

		dep.AlreadyActivated = isActivated

		// Categorize based on dependency type and activation status
		switch dep.DependencyType {
		case "required":
			if !isActivated {
				result.MissingRequiredTools = append(result.MissingRequiredTools, dep)
				result.CanActivate = false
			}
			if dep.AutoInstallOnActivation && !isActivated {
				result.AutoInstallSuggestions = append(result.AutoInstallSuggestions, dep)
			}
		case "recommended":
			if !isActivated {
				result.RecommendedTools = append(result.RecommendedTools, dep)
			}
			if dep.AutoInstallOnActivation && !isActivated {
				result.AutoInstallSuggestions = append(result.AutoInstallSuggestions, dep)
			}
		case "optional":
			if !isActivated {
				result.OptionalTools = append(result.OptionalTools, dep)
			}
		}
	}

	return result, nil
}

// CheckToolDependencies validates all dependencies for a tool activation
func (s *DependencyCheckerService) CheckToolDependencies(ctx context.Context, tenantID, toolID string) (*DependencyCheckResult, error) {
	result := &DependencyCheckResult{
		CanActivate:            true,
		MissingRequiredModules: []DependencyInfo{},
		MissingRequiredTools:   []DependencyInfo{},
		RecommendedModules:     []DependencyInfo{},
		RecommendedTools:       []DependencyInfo{},
		OptionalModules:        []DependencyInfo{},
		OptionalTools:          []DependencyInfo{},
		AutoInstallSuggestions: []DependencyInfo{},
	}

	// Get tool dependencies
	toolDeps, err := s.getToolDependencies(ctx, toolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool dependencies: %w", err)
	}

	// Check each tool dependency
	for _, dep := range toolDeps {
		isActivated, err := s.tenantToolRepo.IsToolActivated(ctx, tenantID, dep.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check tool activation: %w", err)
		}

		dep.AlreadyActivated = isActivated

		// Categorize based on dependency type and activation status
		switch dep.DependencyType {
		case "required":
			if !isActivated {
				result.MissingRequiredTools = append(result.MissingRequiredTools, dep)
				result.CanActivate = false
			}
			if dep.AutoInstallOnActivation && !isActivated {
				result.AutoInstallSuggestions = append(result.AutoInstallSuggestions, dep)
			}
		case "recommended":
			if !isActivated {
				result.RecommendedTools = append(result.RecommendedTools, dep)
			}
			if dep.AutoInstallOnActivation && !isActivated {
				result.AutoInstallSuggestions = append(result.AutoInstallSuggestions, dep)
			}
		case "optional":
			if !isActivated {
				result.OptionalTools = append(result.OptionalTools, dep)
			}
		}
	}

	return result, nil
}

// getModuleDependencies retrieves module dependencies for a module
func (s *DependencyCheckerService) getModuleDependencies(ctx context.Context, moduleID string) ([]DependencyInfo, error) {
	query := `
		SELECT
			md.depends_on_module_id,
			md.dependency_type,
			md.auto_install_on_activation,
			md.install_order,
			m.code,
			m.name
		FROM module_dependencies md
		JOIN modules m ON m.id = md.depends_on_module_id
		WHERE md.module_id = $1
		  AND md.depends_on_module_id IS NOT NULL
		  AND m.deleted_at IS NULL
		ORDER BY md.install_order
	`

	rows, err := s.db.QueryContext(ctx, query, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []DependencyInfo
	for rows.Next() {
		var dep DependencyInfo
		err := rows.Scan(
			&dep.ID,
			&dep.DependencyType,
			&dep.AutoInstallOnActivation,
			&dep.InstallOrder,
			&dep.Code,
			&dep.Name,
		)
		if err != nil {
			return nil, err
		}
		dep.Type = "module"
		deps = append(deps, dep)
	}

	return deps, rows.Err()
}

// getModuleToolDependencies retrieves tool dependencies for a module
func (s *DependencyCheckerService) getModuleToolDependencies(ctx context.Context, moduleID string) ([]DependencyInfo, error) {
	query := `
		SELECT
			md.depends_on_tool_id,
			md.dependency_type,
			md.auto_install_on_activation,
			md.install_order,
			t.code,
			t.name
		FROM module_dependencies md
		JOIN tools t ON t.id = md.depends_on_tool_id
		WHERE md.module_id = $1
		  AND md.depends_on_tool_id IS NOT NULL
		  AND t.deleted_at IS NULL
		ORDER BY md.install_order
	`

	rows, err := s.db.QueryContext(ctx, query, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []DependencyInfo
	for rows.Next() {
		var dep DependencyInfo
		err := rows.Scan(
			&dep.ID,
			&dep.DependencyType,
			&dep.AutoInstallOnActivation,
			&dep.InstallOrder,
			&dep.Code,
			&dep.Name,
		)
		if err != nil {
			return nil, err
		}
		dep.Type = "tool"
		deps = append(deps, dep)
	}

	return deps, rows.Err()
}

// getToolDependencies retrieves tool dependencies for a tool
func (s *DependencyCheckerService) getToolDependencies(ctx context.Context, toolID string) ([]DependencyInfo, error) {
	query := `
		SELECT
			td.depends_on_tool_id,
			td.dependency_type,
			td.auto_install_on_activation,
			td.install_order,
			t.code,
			t.name
		FROM tool_dependencies td
		JOIN tools t ON t.id = td.depends_on_tool_id
		WHERE td.tool_id = $1
		  AND t.deleted_at IS NULL
		ORDER BY td.install_order
	`

	rows, err := s.db.QueryContext(ctx, query, toolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []DependencyInfo
	for rows.Next() {
		var dep DependencyInfo
		err := rows.Scan(
			&dep.ID,
			&dep.DependencyType,
			&dep.AutoInstallOnActivation,
			&dep.InstallOrder,
			&dep.Code,
			&dep.Name,
		)
		if err != nil {
			return nil, err
		}
		dep.Type = "tool"
		deps = append(deps, dep)
	}

	return deps, rows.Err()
}

// GetInstallOrder returns dependencies in correct installation order
func (s *DependencyCheckerService) GetInstallOrder(dependencies []DependencyInfo) []DependencyInfo {
	// Dependencies are already ordered by install_order from database query
	// But we can add topological sort here if circular dependencies need to be handled
	return dependencies
}
