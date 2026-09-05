package registry

import (
	"time"

	"github.com/canakyuz/keystone/internal/domain/registry"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
)

// InstallModuleRequest represents module installation request
type InstallModuleRequest struct {
	ModuleID        string `json:"module_id" validate:"required,uuid"`
	AutoActivate    bool   `json:"auto_activate"`
	AutoInstallDeps bool   `json:"auto_install_deps"`
}

// InstallModuleResponse represents module installation response
type InstallModuleResponse struct {
	TenantModule         TenantModuleResponse `json:"tenant_module"`
	DependencyCheck      DependencyCheckDTO   `json:"dependency_check"`
	AutoInstalledModules []string             `json:"auto_installed_modules,omitempty"`
	AutoInstalledTools   []string             `json:"auto_installed_tools,omitempty"`
}

// InstallToolRequest represents tool installation request
type InstallToolRequest struct {
	ToolID          string `json:"tool_id" validate:"required,uuid"`
	ModuleID        string `json:"module_id,omitempty"`
	AutoActivate    bool   `json:"auto_activate"`
	AutoInstallDeps bool   `json:"auto_install_deps"`
}

// InstallToolResponse represents tool installation response
type InstallToolResponse struct {
	TenantTool         TenantToolResponse `json:"tenant_tool"`
	DependencyCheck    DependencyCheckDTO `json:"dependency_check"`
	AutoInstalledTools []string           `json:"auto_installed_tools,omitempty"`
}

// TenantModuleResponse represents tenant module response
type TenantModuleResponse struct {
	ID                      string                      `json:"id"`
	TenantID                string                      `json:"tenant_id"`
	ModuleID                string                      `json:"module_id"`
	Status                  registry.TenantModuleStatus `json:"status"`
	IsEnabled               bool                        `json:"is_enabled"`
	InstalledAt             time.Time                   `json:"installed_at"`
	ActivatedAt             *time.Time                  `json:"activated_at,omitempty"`
	DeactivatedAt           *time.Time                  `json:"deactivated_at,omitempty"`
	LastUsedAt              *time.Time                  `json:"last_used_at,omitempty"`
	InstalledVersion        string                      `json:"installed_version"`
	LatestCompatibleVersion string                      `json:"latest_compatible_version,omitempty"`
	SubscriptionStatus      registry.SubscriptionStatus `json:"subscription_status,omitempty"`
	SubscriptionStart       *time.Time                  `json:"subscription_start,omitempty"`
	SubscriptionEnd         *time.Time                  `json:"subscription_end,omitempty"`
	TrialEndsAt             *time.Time                  `json:"trial_ends_at,omitempty"`
	NextBillingDate         *time.Time                  `json:"next_billing_date,omitempty"`
	PricingPlan             string                      `json:"pricing_plan,omitempty"`
	BillingCycle            string                      `json:"billing_cycle,omitempty"`
	AmountPaid              float64                     `json:"amount_paid,omitempty"`
	Currency                string                      `json:"currency,omitempty"`
	SetupCompleted          bool                        `json:"setup_completed"`
	OnboardingCompleted     bool                        `json:"onboarding_completed"`
	CreatedAt               time.Time                   `json:"created_at"`
	UpdatedAt               time.Time                   `json:"updated_at"`
}

// TenantToolResponse represents tenant tool response
type TenantToolResponse struct {
	ID                       string                      `json:"id"`
	TenantID                 string                      `json:"tenant_id"`
	ToolID                   string                      `json:"tool_id"`
	ModuleID                 string                      `json:"module_id,omitempty"`
	Status                   registry.TenantToolStatus   `json:"status"`
	IsEnabled                bool                        `json:"is_enabled"`
	InstalledAt              time.Time                   `json:"installed_at"`
	ActivatedAt              *time.Time                  `json:"activated_at,omitempty"`
	DeactivatedAt            *time.Time                  `json:"deactivated_at,omitempty"`
	LastUsedAt               *time.Time                  `json:"last_used_at,omitempty"`
	InstalledVersion         string                      `json:"installed_version"`
	LatestCompatibleVersion  string                      `json:"latest_compatible_version,omitempty"`
	IntegrationEnabled       bool                        `json:"integration_enabled"`
	IntegrationStatus        registry.IntegrationStatus  `json:"integration_status,omitempty"`
	IntegrationVerified      bool                        `json:"integration_verified"`
	IntegrationVerifiedAt    *time.Time                  `json:"integration_verified_at,omitempty"`
	ProviderAccountID        string                      `json:"provider_account_id,omitempty"`
	SubscriptionStatus       registry.SubscriptionStatus `json:"subscription_status,omitempty"`
	SubscriptionStart        *time.Time                  `json:"subscription_start,omitempty"`
	SubscriptionEnd          *time.Time                  `json:"subscription_end,omitempty"`
	TrialEndsAt              *time.Time                  `json:"trial_ends_at,omitempty"`
	NextBillingDate          *time.Time                  `json:"next_billing_date,omitempty"`
	PricingPlan              string                      `json:"pricing_plan,omitempty"`
	BillingCycle             string                      `json:"billing_cycle,omitempty"`
	AmountPaid               float64                     `json:"amount_paid,omitempty"`
	Currency                 string                      `json:"currency,omitempty"`
	TransactionFeesCollected float64                     `json:"transaction_fees_collected,omitempty"`
	SetupCompleted           bool                        `json:"setup_completed"`
	OnboardingCompleted      bool                        `json:"onboarding_completed"`
	HealthStatus             registry.HealthStatus       `json:"health_status"`
	LastHealthCheck          *time.Time                  `json:"last_health_check,omitempty"`
	ErrorCount               int                         `json:"error_count"`
	LastError                string                      `json:"last_error,omitempty"`
	LastErrorAt              *time.Time                  `json:"last_error_at,omitempty"`
	CreatedAt                time.Time                   `json:"created_at"`
	UpdatedAt                time.Time                   `json:"updated_at"`
}

// DependencyCheckDTO represents dependency check result
type DependencyCheckDTO struct {
	CanActivate            bool             `json:"can_activate"`
	MissingRequiredModules []DependencyInfo `json:"missing_required_modules,omitempty"`
	MissingRequiredTools   []DependencyInfo `json:"missing_required_tools,omitempty"`
	RecommendedModules     []DependencyInfo `json:"recommended_modules,omitempty"`
	RecommendedTools       []DependencyInfo `json:"recommended_tools,omitempty"`
	OptionalModules        []DependencyInfo `json:"optional_modules,omitempty"`
	OptionalTools          []DependencyInfo `json:"optional_tools,omitempty"`
	AutoInstallSuggestions []DependencyInfo `json:"auto_install_suggestions,omitempty"`
}

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

// ToTenantModuleResponse converts domain tenant module to response
func ToTenantModuleResponse(tm *registry.TenantModule) TenantModuleResponse {
	return TenantModuleResponse{
		ID:                      tm.ID,
		TenantID:                tm.TenantID,
		ModuleID:                tm.ModuleID,
		Status:                  tm.Status,
		IsEnabled:               tm.IsEnabled,
		InstalledAt:             tm.InstalledAt,
		ActivatedAt:             tm.ActivatedAt,
		DeactivatedAt:           tm.DeactivatedAt,
		LastUsedAt:              tm.LastUsedAt,
		InstalledVersion:        tm.InstalledVersion,
		LatestCompatibleVersion: tm.LatestCompatibleVersion,
		SubscriptionStatus:      tm.SubscriptionStatus,
		SubscriptionStart:       tm.SubscriptionStart,
		SubscriptionEnd:         tm.SubscriptionEnd,
		TrialEndsAt:             tm.TrialEndsAt,
		NextBillingDate:         tm.NextBillingDate,
		PricingPlan:             tm.PricingPlan,
		BillingCycle:            tm.BillingCycle,
		AmountPaid:              tm.AmountPaid,
		Currency:                tm.Currency,
		SetupCompleted:          tm.SetupCompleted,
		OnboardingCompleted:     tm.OnboardingCompleted,
		CreatedAt:               tm.CreatedAt,
		UpdatedAt:               tm.UpdatedAt,
	}
}

// ToTenantToolResponse converts domain tenant tool to response
func ToTenantToolResponse(tt *registry.TenantTool) TenantToolResponse {
	return TenantToolResponse{
		ID:                       tt.ID,
		TenantID:                 tt.TenantID,
		ToolID:                   tt.ToolID,
		ModuleID:                 tt.ModuleID,
		Status:                   tt.Status,
		IsEnabled:                tt.IsEnabled,
		InstalledAt:              tt.InstalledAt,
		ActivatedAt:              tt.ActivatedAt,
		DeactivatedAt:            tt.DeactivatedAt,
		LastUsedAt:               tt.LastUsedAt,
		InstalledVersion:         tt.InstalledVersion,
		LatestCompatibleVersion:  tt.LatestCompatibleVersion,
		IntegrationEnabled:       tt.IntegrationEnabled,
		IntegrationStatus:        tt.IntegrationStatus,
		IntegrationVerified:      tt.IntegrationVerified,
		IntegrationVerifiedAt:    tt.IntegrationVerifiedAt,
		ProviderAccountID:        tt.ProviderAccountID,
		SubscriptionStatus:       tt.SubscriptionStatus,
		SubscriptionStart:        tt.SubscriptionStart,
		SubscriptionEnd:          tt.SubscriptionEnd,
		TrialEndsAt:              tt.TrialEndsAt,
		NextBillingDate:          tt.NextBillingDate,
		PricingPlan:              tt.PricingPlan,
		BillingCycle:             tt.BillingCycle,
		AmountPaid:               tt.AmountPaid,
		Currency:                 tt.Currency,
		TransactionFeesCollected: tt.TransactionFeesCollected,
		SetupCompleted:           tt.SetupCompleted,
		OnboardingCompleted:      tt.OnboardingCompleted,
		HealthStatus:             tt.HealthStatus,
		LastHealthCheck:          tt.LastHealthCheck,
		ErrorCount:               tt.ErrorCount,
		LastError:                tt.LastError,
		LastErrorAt:              tt.LastErrorAt,
		CreatedAt:                tt.CreatedAt,
		UpdatedAt:                tt.UpdatedAt,
	}
}

// ToDependencyCheckDTO converts service dependency check result to DTO
func ToDependencyCheckDTO(result *registryService.DependencyCheckResult) DependencyCheckDTO {
	return DependencyCheckDTO{
		CanActivate:            result.CanActivate,
		MissingRequiredModules: toDependencyInfoList(result.MissingRequiredModules),
		MissingRequiredTools:   toDependencyInfoList(result.MissingRequiredTools),
		RecommendedModules:     toDependencyInfoList(result.RecommendedModules),
		RecommendedTools:       toDependencyInfoList(result.RecommendedTools),
		OptionalModules:        toDependencyInfoList(result.OptionalModules),
		OptionalTools:          toDependencyInfoList(result.OptionalTools),
		AutoInstallSuggestions: toDependencyInfoList(result.AutoInstallSuggestions),
	}
}

func toDependencyInfoList(deps []registryService.DependencyInfo) []DependencyInfo {
	result := make([]DependencyInfo, len(deps))
	for i, dep := range deps {
		result[i] = DependencyInfo{
			Type:                    dep.Type,
			ID:                      dep.ID,
			Code:                    dep.Code,
			Name:                    dep.Name,
			DependencyType:          dep.DependencyType,
			AutoInstallOnActivation: dep.AutoInstallOnActivation,
			InstallOrder:            dep.InstallOrder,
			AlreadyActivated:        dep.AlreadyActivated,
		}
	}
	return result
}
