package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nexpaces-api/internal/domain/registry"
)

type tenantToolRepository struct {
	db *sql.DB
}

// NewTenantToolRepository creates a new tenant tool repository
func NewTenantToolRepository(db *sql.DB) registry.TenantToolRepository {
	return &tenantToolRepository{db: db}
}

// Create creates a new tenant tool activation
func (r *tenantToolRepository) Create(ctx context.Context, tenantTool *registry.TenantTool) error {
	query := `
		INSERT INTO tenant_tools (
			id, tenant_id, tool_id, module_id, status, is_enabled,
			installed_at, activated_at, deactivated_at, last_used_at,
			installed_version, latest_compatible_version,
			configuration, api_keys, webhook_config,
			integration_enabled, integration_status, integration_verified, integration_verified_at, provider_account_id,
			limits, current_usage, rate_limits,
			subscription_status, subscription_start, subscription_end, trial_ends_at, next_billing_date,
			pricing_plan, billing_cycle, amount_paid, currency, transaction_fees_collected,
			setup_completed, setup_steps_completed, onboarding_completed,
			allowed_roles, restricted_features,
			health_status, last_health_check, error_count, last_error, last_error_at,
			notes, metadata,
			created_at, updated_at, created_by, updated_by, activated_by, deactivated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33,
			$34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		tenantTool.ID, tenantTool.TenantID, tenantTool.ToolID, tenantTool.ModuleID, tenantTool.Status, tenantTool.IsEnabled,
		tenantTool.InstalledAt, tenantTool.ActivatedAt, tenantTool.DeactivatedAt, tenantTool.LastUsedAt,
		tenantTool.InstalledVersion, tenantTool.LatestCompatibleVersion,
		tenantTool.Configuration, tenantTool.APIKeys, tenantTool.WebhookConfig,
		tenantTool.IntegrationEnabled, tenantTool.IntegrationStatus, tenantTool.IntegrationVerified, tenantTool.IntegrationVerifiedAt, tenantTool.ProviderAccountID,
		tenantTool.Limits, tenantTool.CurrentUsage, tenantTool.RateLimits,
		tenantTool.SubscriptionStatus, tenantTool.SubscriptionStart, tenantTool.SubscriptionEnd, tenantTool.TrialEndsAt, tenantTool.NextBillingDate,
		tenantTool.PricingPlan, tenantTool.BillingCycle, tenantTool.AmountPaid, tenantTool.Currency, tenantTool.TransactionFeesCollected,
		tenantTool.SetupCompleted, tenantTool.SetupStepsCompleted, tenantTool.OnboardingCompleted,
		tenantTool.AllowedRoles, tenantTool.RestrictedFeatures,
		tenantTool.HealthStatus, tenantTool.LastHealthCheck, tenantTool.ErrorCount, tenantTool.LastError, tenantTool.LastErrorAt,
		tenantTool.Notes, tenantTool.Metadata,
		tenantTool.CreatedAt, tenantTool.UpdatedAt, tenantTool.CreatedBy, tenantTool.UpdatedBy, tenantTool.ActivatedBy, tenantTool.DeactivatedBy,
	)

	return err
}

// GetByID retrieves a tenant tool by ID
func (r *tenantToolRepository) GetByID(ctx context.Context, id string) (*registry.TenantTool, error) {
	query := `
		SELECT id, tenant_id, tool_id, module_id, status, is_enabled,
			installed_at, activated_at, deactivated_at, last_used_at,
			installed_version, latest_compatible_version,
			configuration, api_keys, webhook_config,
			integration_enabled, integration_status, integration_verified, integration_verified_at, provider_account_id,
			limits, current_usage, rate_limits,
			subscription_status, subscription_start, subscription_end, trial_ends_at, next_billing_date,
			pricing_plan, billing_cycle, amount_paid, currency, transaction_fees_collected,
			setup_completed, setup_steps_completed, onboarding_completed,
			allowed_roles, restricted_features,
			health_status, last_health_check, error_count, last_error, last_error_at,
			notes, metadata,
			created_at, updated_at, deleted_at, created_by, updated_by, activated_by, deactivated_by
		FROM tenant_tools
		WHERE id = $1 AND deleted_at IS NULL
	`

	tt := &registry.TenantTool{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tt.ID, &tt.TenantID, &tt.ToolID, &tt.ModuleID, &tt.Status, &tt.IsEnabled,
		&tt.InstalledAt, &tt.ActivatedAt, &tt.DeactivatedAt, &tt.LastUsedAt,
		&tt.InstalledVersion, &tt.LatestCompatibleVersion,
		&tt.Configuration, &tt.APIKeys, &tt.WebhookConfig,
		&tt.IntegrationEnabled, &tt.IntegrationStatus, &tt.IntegrationVerified, &tt.IntegrationVerifiedAt, &tt.ProviderAccountID,
		&tt.Limits, &tt.CurrentUsage, &tt.RateLimits,
		&tt.SubscriptionStatus, &tt.SubscriptionStart, &tt.SubscriptionEnd, &tt.TrialEndsAt, &tt.NextBillingDate,
		&tt.PricingPlan, &tt.BillingCycle, &tt.AmountPaid, &tt.Currency, &tt.TransactionFeesCollected,
		&tt.SetupCompleted, &tt.SetupStepsCompleted, &tt.OnboardingCompleted,
		&tt.AllowedRoles, &tt.RestrictedFeatures,
		&tt.HealthStatus, &tt.LastHealthCheck, &tt.ErrorCount, &tt.LastError, &tt.LastErrorAt,
		&tt.Notes, &tt.Metadata,
		&tt.CreatedAt, &tt.UpdatedAt, &tt.DeletedAt, &tt.CreatedBy, &tt.UpdatedBy, &tt.ActivatedBy, &tt.DeactivatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrTenantToolNotFound
	}

	return tt, err
}

// GetByTenantAndTool retrieves a tenant tool by tenant and tool IDs
func (r *tenantToolRepository) GetByTenantAndTool(ctx context.Context, tenantID, toolID string) (*registry.TenantTool, error) {
	query := `
		SELECT id, tenant_id, tool_id, module_id, status, is_enabled,
			installed_at, activated_at, deactivated_at, last_used_at,
			installed_version, latest_compatible_version,
			configuration, api_keys, webhook_config,
			integration_enabled, integration_status, integration_verified, integration_verified_at, provider_account_id,
			limits, current_usage, rate_limits,
			subscription_status, subscription_start, subscription_end, trial_ends_at, next_billing_date,
			pricing_plan, billing_cycle, amount_paid, currency, transaction_fees_collected,
			setup_completed, setup_steps_completed, onboarding_completed,
			allowed_roles, restricted_features,
			health_status, last_health_check, error_count, last_error, last_error_at,
			notes, metadata,
			created_at, updated_at, deleted_at, created_by, updated_by, activated_by, deactivated_by
		FROM tenant_tools
		WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL
	`

	tt := &registry.TenantTool{}
	err := r.db.QueryRowContext(ctx, query, tenantID, toolID).Scan(
		&tt.ID, &tt.TenantID, &tt.ToolID, &tt.ModuleID, &tt.Status, &tt.IsEnabled,
		&tt.InstalledAt, &tt.ActivatedAt, &tt.DeactivatedAt, &tt.LastUsedAt,
		&tt.InstalledVersion, &tt.LatestCompatibleVersion,
		&tt.Configuration, &tt.APIKeys, &tt.WebhookConfig,
		&tt.IntegrationEnabled, &tt.IntegrationStatus, &tt.IntegrationVerified, &tt.IntegrationVerifiedAt, &tt.ProviderAccountID,
		&tt.Limits, &tt.CurrentUsage, &tt.RateLimits,
		&tt.SubscriptionStatus, &tt.SubscriptionStart, &tt.SubscriptionEnd, &tt.TrialEndsAt, &tt.NextBillingDate,
		&tt.PricingPlan, &tt.BillingCycle, &tt.AmountPaid, &tt.Currency, &tt.TransactionFeesCollected,
		&tt.SetupCompleted, &tt.SetupStepsCompleted, &tt.OnboardingCompleted,
		&tt.AllowedRoles, &tt.RestrictedFeatures,
		&tt.HealthStatus, &tt.LastHealthCheck, &tt.ErrorCount, &tt.LastError, &tt.LastErrorAt,
		&tt.Notes, &tt.Metadata,
		&tt.CreatedAt, &tt.UpdatedAt, &tt.DeletedAt, &tt.CreatedBy, &tt.UpdatedBy, &tt.ActivatedBy, &tt.DeactivatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrTenantToolNotFound
	}

	return tt, err
}

// Update updates a tenant tool
func (r *tenantToolRepository) Update(ctx context.Context, tenantTool *registry.TenantTool) error {
	query := `
		UPDATE tenant_tools SET
			status = $2, is_enabled = $3,
			activated_at = $4, deactivated_at = $5, last_used_at = $6,
			latest_compatible_version = $7,
			configuration = $8, api_keys = $9, webhook_config = $10,
			integration_enabled = $11, integration_status = $12, integration_verified = $13, integration_verified_at = $14, provider_account_id = $15,
			limits = $16, current_usage = $17, rate_limits = $18,
			subscription_status = $19, subscription_start = $20, subscription_end = $21, trial_ends_at = $22, next_billing_date = $23,
			pricing_plan = $24, billing_cycle = $25, amount_paid = $26, transaction_fees_collected = $27,
			setup_completed = $28, setup_steps_completed = $29, onboarding_completed = $30,
			allowed_roles = $31, restricted_features = $32,
			health_status = $33, last_health_check = $34, error_count = $35, last_error = $36, last_error_at = $37,
			notes = $38, metadata = $39,
			updated_at = $40, updated_by = $41, activated_by = $42, deactivated_by = $43
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query,
		tenantTool.ID, tenantTool.Status, tenantTool.IsEnabled,
		tenantTool.ActivatedAt, tenantTool.DeactivatedAt, tenantTool.LastUsedAt,
		tenantTool.LatestCompatibleVersion,
		tenantTool.Configuration, tenantTool.APIKeys, tenantTool.WebhookConfig,
		tenantTool.IntegrationEnabled, tenantTool.IntegrationStatus, tenantTool.IntegrationVerified, tenantTool.IntegrationVerifiedAt, tenantTool.ProviderAccountID,
		tenantTool.Limits, tenantTool.CurrentUsage, tenantTool.RateLimits,
		tenantTool.SubscriptionStatus, tenantTool.SubscriptionStart, tenantTool.SubscriptionEnd, tenantTool.TrialEndsAt, tenantTool.NextBillingDate,
		tenantTool.PricingPlan, tenantTool.BillingCycle, tenantTool.AmountPaid, tenantTool.TransactionFeesCollected,
		tenantTool.SetupCompleted, tenantTool.SetupStepsCompleted, tenantTool.OnboardingCompleted,
		tenantTool.AllowedRoles, tenantTool.RestrictedFeatures,
		tenantTool.HealthStatus, tenantTool.LastHealthCheck, tenantTool.ErrorCount, tenantTool.LastError, tenantTool.LastErrorAt,
		tenantTool.Notes, tenantTool.Metadata,
		tenantTool.UpdatedAt, tenantTool.UpdatedBy, tenantTool.ActivatedBy, tenantTool.DeactivatedBy,
	)

	return err
}

// Delete soft deletes a tenant tool
func (r *tenantToolRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE tenant_tools SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByTenant retrieves tenant tools with filters
func (r *tenantToolRepository) ListByTenant(ctx context.Context, tenantID string, filters registry.TenantToolFilters) ([]*registry.TenantTool, error) {
	query := `
		SELECT id, tenant_id, tool_id, module_id, status, is_enabled,
			installed_at, activated_at, last_used_at,
			installed_version, latest_compatible_version,
			integration_enabled, integration_status, integration_verified, provider_account_id,
			subscription_status, pricing_plan, billing_cycle,
			setup_completed, onboarding_completed,
			health_status, last_health_check, error_count,
			created_at, updated_at
		FROM tenant_tools
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	conditions, args := r.buildFilterConditions(filters)
	allArgs := append([]interface{}{tenantID}, args...)

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += r.buildOrderBy(filters)
	query += r.buildPagination(filters)

	rows, err := r.db.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenantTools []*registry.TenantTool
	for rows.Next() {
		tt := &registry.TenantTool{}
		err := rows.Scan(
			&tt.ID, &tt.TenantID, &tt.ToolID, &tt.ModuleID, &tt.Status, &tt.IsEnabled,
			&tt.InstalledAt, &tt.ActivatedAt, &tt.LastUsedAt,
			&tt.InstalledVersion, &tt.LatestCompatibleVersion,
			&tt.IntegrationEnabled, &tt.IntegrationStatus, &tt.IntegrationVerified, &tt.ProviderAccountID,
			&tt.SubscriptionStatus, &tt.PricingPlan, &tt.BillingCycle,
			&tt.SetupCompleted, &tt.OnboardingCompleted,
			&tt.HealthStatus, &tt.LastHealthCheck, &tt.ErrorCount,
			&tt.CreatedAt, &tt.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tenantTools = append(tenantTools, tt)
	}

	return tenantTools, rows.Err()
}

// ListActivatedByTenant retrieves activated tools for a tenant
func (r *tenantToolRepository) ListActivatedByTenant(ctx context.Context, tenantID string) ([]*registry.TenantTool, error) {
	filters := registry.TenantToolFilters{
		Status:    tenantToolStatusPtr(registry.TenantToolStatusActive),
		IsEnabled: boolPtr(true),
		SortBy:    "activated_at",
		SortOrder: "desc",
	}
	return r.ListByTenant(ctx, tenantID, filters)
}

// IsToolActivated checks if a tool is activated for a tenant
func (r *tenantToolRepository) IsToolActivated(ctx context.Context, tenantID, toolID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenant_tools
			WHERE tenant_id = $1 AND tool_id = $2
			  AND status = 'active' AND is_enabled = true
			  AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, toolID).Scan(&exists)
	return exists, err
}

// Activate activates a tool for a tenant
func (r *tenantToolRepository) Activate(ctx context.Context, tenantID, toolID, activatedBy string) error {
	// First, get the existing tenant tool
	tt, err := r.GetByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return err
	}

	// Activate using domain logic
	if err := tt.Activate(activatedBy); err != nil {
		return err
	}

	// Update in database
	return r.Update(ctx, tt)
}

// Deactivate deactivates a tool for a tenant
func (r *tenantToolRepository) Deactivate(ctx context.Context, tenantID, toolID, deactivatedBy string) error {
	// First, get the existing tenant tool
	tt, err := r.GetByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return err
	}

	// Deactivate using domain logic
	if err := tt.Deactivate(deactivatedBy); err != nil {
		return err
	}

	// Update in database
	return r.Update(ctx, tt)
}

// VerifyIntegration marks integration as verified
func (r *tenantToolRepository) VerifyIntegration(ctx context.Context, tenantID, toolID string) error {
	// First, get the existing tenant tool
	tt, err := r.GetByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return err
	}

	// Verify using domain logic
	tt.VerifyIntegration()

	// Update in database
	return r.Update(ctx, tt)
}

// UpdateIntegrationStatus updates the integration status
func (r *tenantToolRepository) UpdateIntegrationStatus(ctx context.Context, tenantID, toolID string, status registry.IntegrationStatus) error {
	query := `
		UPDATE tenant_tools
		SET integration_status = $3, updated_at = NOW()
		WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, tenantID, toolID, status)
	return err
}

// UpdateHealthStatus updates the health status
func (r *tenantToolRepository) UpdateHealthStatus(ctx context.Context, tenantID, toolID string, status registry.HealthStatus) error {
	// First, get the existing tenant tool
	tt, err := r.GetByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return err
	}

	// Update health status using domain logic
	tt.SetHealthStatus(status)

	// Update in database
	return r.Update(ctx, tt)
}

// RecordError records an error for a tool
func (r *tenantToolRepository) RecordError(ctx context.Context, tenantID, toolID, errorMsg string) error {
	// First, get the existing tenant tool
	tt, err := r.GetByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return err
	}

	// Record error using domain logic
	tt.RecordError(errorMsg)

	// Update in database
	return r.Update(ctx, tt)
}

// UpdateLastUsed updates the last used timestamp
func (r *tenantToolRepository) UpdateLastUsed(ctx context.Context, tenantID, toolID string) error {
	query := `
		UPDATE tenant_tools
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, tenantID, toolID)
	return err
}

// UpdateUsage updates the current usage for a tool
func (r *tenantToolRepository) UpdateUsage(ctx context.Context, tenantID, toolID string, usage map[string]interface{}) error {
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	query := `
		UPDATE tenant_tools
		SET current_usage = $3, updated_at = NOW()
		WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL
	`

	_, err = r.db.ExecContext(ctx, query, tenantID, toolID, usageJSON)
	return err
}

// Helper functions
func (r *tenantToolRepository) buildFilterConditions(filters registry.TenantToolFilters) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	paramCount := 2 // Start at 2 because $1 is tenantID

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", paramCount))
		args = append(args, *filters.Status)
		paramCount++
	}

	if filters.IsEnabled != nil {
		conditions = append(conditions, fmt.Sprintf("is_enabled = $%d", paramCount))
		args = append(args, *filters.IsEnabled)
		paramCount++
	}

	if filters.IntegrationStatus != nil {
		conditions = append(conditions, fmt.Sprintf("integration_status = $%d", paramCount))
		args = append(args, *filters.IntegrationStatus)
		paramCount++
	}

	if filters.HealthStatus != nil {
		conditions = append(conditions, fmt.Sprintf("health_status = $%d", paramCount))
		args = append(args, *filters.HealthStatus)
		paramCount++
	}

	if filters.SubscriptionStatus != nil {
		conditions = append(conditions, fmt.Sprintf("subscription_status = $%d", paramCount))
		args = append(args, *filters.SubscriptionStatus)
		paramCount++
	}

	if filters.SetupCompleted != nil {
		conditions = append(conditions, fmt.Sprintf("setup_completed = $%d", paramCount))
		args = append(args, *filters.SetupCompleted)
		paramCount++
	}

	return conditions, args
}

func (r *tenantToolRepository) buildOrderBy(filters registry.TenantToolFilters) string {
	sortBy := "installed_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)
}

func (r *tenantToolRepository) buildPagination(filters registry.TenantToolFilters) string {
	limit := 50
	if filters.Limit > 0 {
		limit = filters.Limit
	}

	offset := 0
	if filters.Offset > 0 {
		offset = filters.Offset
	}

	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

func tenantToolStatusPtr(s registry.TenantToolStatus) *registry.TenantToolStatus {
	return &s
}
