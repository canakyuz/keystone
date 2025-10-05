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

type tenantModuleRepository struct {
	db *sql.DB
}

// NewTenantModuleRepository creates a new tenant module repository
func NewTenantModuleRepository(db *sql.DB) registry.TenantModuleRepository {
	return &tenantModuleRepository{db: db}
}

// Create creates a new tenant module activation
func (r *tenantModuleRepository) Create(ctx context.Context, tenantModule *registry.TenantModule) error {
	query := `
		INSERT INTO tenant_modules (
			id, tenant_id, module_id, status, is_enabled,
			installed_at, activated_at, deactivated_at, last_used_at,
			installed_version, latest_compatible_version,
			configuration, features_enabled, limits, current_usage,
			subscription_status, subscription_start, subscription_end, trial_ends_at, next_billing_date,
			pricing_plan, billing_cycle, amount_paid, currency,
			setup_completed, setup_steps_completed, onboarding_completed,
			allowed_roles, restricted_features,
			notes, metadata,
			created_at, updated_at, created_by, updated_by, activated_by, deactivated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		tenantModule.ID, tenantModule.TenantID, tenantModule.ModuleID, tenantModule.Status, tenantModule.IsEnabled,
		tenantModule.InstalledAt, tenantModule.ActivatedAt, tenantModule.DeactivatedAt, tenantModule.LastUsedAt,
		tenantModule.InstalledVersion, tenantModule.LatestCompatibleVersion,
		tenantModule.Configuration, tenantModule.FeaturesEnabled, tenantModule.Limits, tenantModule.CurrentUsage,
		tenantModule.SubscriptionStatus, tenantModule.SubscriptionStart, tenantModule.SubscriptionEnd, tenantModule.TrialEndsAt, tenantModule.NextBillingDate,
		tenantModule.PricingPlan, tenantModule.BillingCycle, tenantModule.AmountPaid, tenantModule.Currency,
		tenantModule.SetupCompleted, tenantModule.SetupStepsCompleted, tenantModule.OnboardingCompleted,
		tenantModule.AllowedRoles, tenantModule.RestrictedFeatures,
		tenantModule.Notes, tenantModule.Metadata,
		tenantModule.CreatedAt, tenantModule.UpdatedAt, tenantModule.CreatedBy, tenantModule.UpdatedBy, tenantModule.ActivatedBy, tenantModule.DeactivatedBy,
	)

	return err
}

// GetByID retrieves a tenant module by ID
func (r *tenantModuleRepository) GetByID(ctx context.Context, id string) (*registry.TenantModule, error) {
	query := `
		SELECT id, tenant_id, module_id, status, is_enabled,
			installed_at, activated_at, deactivated_at, last_used_at,
			installed_version, latest_compatible_version,
			configuration, features_enabled, limits, current_usage,
			subscription_status, subscription_start, subscription_end, trial_ends_at, next_billing_date,
			pricing_plan, billing_cycle, amount_paid, currency,
			setup_completed, setup_steps_completed, onboarding_completed,
			allowed_roles, restricted_features,
			notes, metadata,
			created_at, updated_at, deleted_at, created_by, updated_by, activated_by, deactivated_by
		FROM tenant_modules
		WHERE id = $1 AND deleted_at IS NULL
	`

	tm := &registry.TenantModule{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tm.ID, &tm.TenantID, &tm.ModuleID, &tm.Status, &tm.IsEnabled,
		&tm.InstalledAt, &tm.ActivatedAt, &tm.DeactivatedAt, &tm.LastUsedAt,
		&tm.InstalledVersion, &tm.LatestCompatibleVersion,
		&tm.Configuration, &tm.FeaturesEnabled, &tm.Limits, &tm.CurrentUsage,
		&tm.SubscriptionStatus, &tm.SubscriptionStart, &tm.SubscriptionEnd, &tm.TrialEndsAt, &tm.NextBillingDate,
		&tm.PricingPlan, &tm.BillingCycle, &tm.AmountPaid, &tm.Currency,
		&tm.SetupCompleted, &tm.SetupStepsCompleted, &tm.OnboardingCompleted,
		&tm.AllowedRoles, &tm.RestrictedFeatures,
		&tm.Notes, &tm.Metadata,
		&tm.CreatedAt, &tm.UpdatedAt, &tm.DeletedAt, &tm.CreatedBy, &tm.UpdatedBy, &tm.ActivatedBy, &tm.DeactivatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrTenantModuleNotFound
	}

	return tm, err
}

// GetByTenantAndModule retrieves a tenant module by tenant and module IDs
func (r *tenantModuleRepository) GetByTenantAndModule(ctx context.Context, tenantID, moduleID string) (*registry.TenantModule, error) {
	query := `
		SELECT id, tenant_id, module_id, status, is_enabled,
			installed_at, activated_at, deactivated_at, last_used_at,
			installed_version, latest_compatible_version,
			configuration, features_enabled, limits, current_usage,
			subscription_status, subscription_start, subscription_end, trial_ends_at, next_billing_date,
			pricing_plan, billing_cycle, amount_paid, currency,
			setup_completed, setup_steps_completed, onboarding_completed,
			allowed_roles, restricted_features,
			notes, metadata,
			created_at, updated_at, deleted_at, created_by, updated_by, activated_by, deactivated_by
		FROM tenant_modules
		WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL
	`

	tm := &registry.TenantModule{}
	err := r.db.QueryRowContext(ctx, query, tenantID, moduleID).Scan(
		&tm.ID, &tm.TenantID, &tm.ModuleID, &tm.Status, &tm.IsEnabled,
		&tm.InstalledAt, &tm.ActivatedAt, &tm.DeactivatedAt, &tm.LastUsedAt,
		&tm.InstalledVersion, &tm.LatestCompatibleVersion,
		&tm.Configuration, &tm.FeaturesEnabled, &tm.Limits, &tm.CurrentUsage,
		&tm.SubscriptionStatus, &tm.SubscriptionStart, &tm.SubscriptionEnd, &tm.TrialEndsAt, &tm.NextBillingDate,
		&tm.PricingPlan, &tm.BillingCycle, &tm.AmountPaid, &tm.Currency,
		&tm.SetupCompleted, &tm.SetupStepsCompleted, &tm.OnboardingCompleted,
		&tm.AllowedRoles, &tm.RestrictedFeatures,
		&tm.Notes, &tm.Metadata,
		&tm.CreatedAt, &tm.UpdatedAt, &tm.DeletedAt, &tm.CreatedBy, &tm.UpdatedBy, &tm.ActivatedBy, &tm.DeactivatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrTenantModuleNotFound
	}

	return tm, err
}

// Update updates a tenant module
func (r *tenantModuleRepository) Update(ctx context.Context, tenantModule *registry.TenantModule) error {
	query := `
		UPDATE tenant_modules SET
			status = $2, is_enabled = $3,
			activated_at = $4, deactivated_at = $5, last_used_at = $6,
			latest_compatible_version = $7,
			configuration = $8, features_enabled = $9, limits = $10, current_usage = $11,
			subscription_status = $12, subscription_start = $13, subscription_end = $14, trial_ends_at = $15, next_billing_date = $16,
			pricing_plan = $17, billing_cycle = $18, amount_paid = $19,
			setup_completed = $20, setup_steps_completed = $21, onboarding_completed = $22,
			allowed_roles = $23, restricted_features = $24,
			notes = $25, metadata = $26,
			updated_at = $27, updated_by = $28, activated_by = $29, deactivated_by = $30
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query,
		tenantModule.ID, tenantModule.Status, tenantModule.IsEnabled,
		tenantModule.ActivatedAt, tenantModule.DeactivatedAt, tenantModule.LastUsedAt,
		tenantModule.LatestCompatibleVersion,
		tenantModule.Configuration, tenantModule.FeaturesEnabled, tenantModule.Limits, tenantModule.CurrentUsage,
		tenantModule.SubscriptionStatus, tenantModule.SubscriptionStart, tenantModule.SubscriptionEnd, tenantModule.TrialEndsAt, tenantModule.NextBillingDate,
		tenantModule.PricingPlan, tenantModule.BillingCycle, tenantModule.AmountPaid,
		tenantModule.SetupCompleted, tenantModule.SetupStepsCompleted, tenantModule.OnboardingCompleted,
		tenantModule.AllowedRoles, tenantModule.RestrictedFeatures,
		tenantModule.Notes, tenantModule.Metadata,
		tenantModule.UpdatedAt, tenantModule.UpdatedBy, tenantModule.ActivatedBy, tenantModule.DeactivatedBy,
	)

	return err
}

// Delete soft deletes a tenant module
func (r *tenantModuleRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE tenant_modules SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByTenant retrieves tenant modules with filters
func (r *tenantModuleRepository) ListByTenant(ctx context.Context, tenantID string, filters registry.TenantModuleFilters) ([]*registry.TenantModule, error) {
	query := `
		SELECT id, tenant_id, module_id, status, is_enabled,
			installed_at, activated_at, last_used_at,
			installed_version, latest_compatible_version,
			subscription_status, pricing_plan, billing_cycle,
			setup_completed, onboarding_completed,
			created_at, updated_at
		FROM tenant_modules
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

	var tenantModules []*registry.TenantModule
	for rows.Next() {
		tm := &registry.TenantModule{}
		err := rows.Scan(
			&tm.ID, &tm.TenantID, &tm.ModuleID, &tm.Status, &tm.IsEnabled,
			&tm.InstalledAt, &tm.ActivatedAt, &tm.LastUsedAt,
			&tm.InstalledVersion, &tm.LatestCompatibleVersion,
			&tm.SubscriptionStatus, &tm.PricingPlan, &tm.BillingCycle,
			&tm.SetupCompleted, &tm.OnboardingCompleted,
			&tm.CreatedAt, &tm.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tenantModules = append(tenantModules, tm)
	}

	return tenantModules, rows.Err()
}

// ListActivatedByTenant retrieves activated modules for a tenant
func (r *tenantModuleRepository) ListActivatedByTenant(ctx context.Context, tenantID string) ([]*registry.TenantModule, error) {
	filters := registry.TenantModuleFilters{
		Status:    tenantModuleStatusPtr(registry.TenantModuleStatusActive),
		IsEnabled: boolPtr(true),
		SortBy:    "activated_at",
		SortOrder: "desc",
	}
	return r.ListByTenant(ctx, tenantID, filters)
}

// IsModuleActivated checks if a module is activated for a tenant
func (r *tenantModuleRepository) IsModuleActivated(ctx context.Context, tenantID, moduleID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenant_modules
			WHERE tenant_id = $1 AND module_id = $2
			  AND status = 'active' AND is_enabled = true
			  AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, moduleID).Scan(&exists)
	return exists, err
}

// Activate activates a module for a tenant
func (r *tenantModuleRepository) Activate(ctx context.Context, tenantID, moduleID, activatedBy string) error {
	// First, get the existing tenant module
	tm, err := r.GetByTenantAndModule(ctx, tenantID, moduleID)
	if err != nil {
		return err
	}

	// Activate using domain logic
	if err := tm.Activate(activatedBy); err != nil {
		return err
	}

	// Update in database
	return r.Update(ctx, tm)
}

// Deactivate deactivates a module for a tenant
func (r *tenantModuleRepository) Deactivate(ctx context.Context, tenantID, moduleID, deactivatedBy string) error {
	// First, get the existing tenant module
	tm, err := r.GetByTenantAndModule(ctx, tenantID, moduleID)
	if err != nil {
		return err
	}

	// Deactivate using domain logic
	if err := tm.Deactivate(deactivatedBy); err != nil {
		return err
	}

	// Update in database
	return r.Update(ctx, tm)
}

// UpdateLastUsed updates the last used timestamp
func (r *tenantModuleRepository) UpdateLastUsed(ctx context.Context, tenantID, moduleID string) error {
	query := `
		UPDATE tenant_modules
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, tenantID, moduleID)
	return err
}

// UpdateUsage updates the current usage for a module
func (r *tenantModuleRepository) UpdateUsage(ctx context.Context, tenantID, moduleID string, usage map[string]interface{}) error {
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	query := `
		UPDATE tenant_modules
		SET current_usage = $3, updated_at = NOW()
		WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL
	`

	_, err = r.db.ExecContext(ctx, query, tenantID, moduleID, usageJSON)
	return err
}

// Helper functions
func (r *tenantModuleRepository) buildFilterConditions(filters registry.TenantModuleFilters) ([]string, []interface{}) {
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

func (r *tenantModuleRepository) buildOrderBy(filters registry.TenantModuleFilters) string {
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

func (r *tenantModuleRepository) buildPagination(filters registry.TenantModuleFilters) string {
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

func tenantModuleStatusPtr(s registry.TenantModuleStatus) *registry.TenantModuleStatus {
	return &s
}
