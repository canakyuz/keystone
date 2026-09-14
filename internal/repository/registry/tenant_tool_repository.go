package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/canakyuz/keystone/internal/domain/registry"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

type tenantToolRepository struct {
	db    *sql.DB
	trail *auditrepo.Repository
}

// NewTenantToolRepository creates a new tenant tool repository.
//
// Every method runs inside a transaction scoped to the tenant it is given; see inTenant.
// Every change made to an installation goes to the trail in that same transaction, so a
// repository built without one refuses to change anything.
func NewTenantToolRepository(db *sql.DB, trail *auditrepo.Repository) registry.TenantToolRepository {
	return &tenantToolRepository{db: db, trail: trail}
}

// tenantToolColumns selects an installation in the order scanTenantTool reads it, on the
// rules of moduleColumns.
//
// api_keys and webhook_config are not selected. They hold a tenant's credentials for the
// provider behind the tool, no caller of this repository needs them, and a value that is
// never read cannot end up in a response or a log line.
const tenantToolColumns = `
	id, tenant_id, tool_id, COALESCE(module_id::text, ''), status, is_enabled,
	installed_at, activated_at, deactivated_at, last_used_at,
	installed_version, COALESCE(latest_compatible_version, ''),
	configuration,
	COALESCE(integration_enabled, FALSE), COALESCE(integration_status, ''), COALESCE(integration_verified, FALSE),
	integration_verified_at, COALESCE(provider_account_id, ''),
	limits, current_usage, rate_limits,
	COALESCE(subscription_status, ''), subscription_start, subscription_end, trial_ends_at, next_billing_date,
	COALESCE(pricing_plan, ''), COALESCE(billing_cycle, ''), COALESCE(amount_paid, 0), COALESCE(currency, ''),
	COALESCE(transaction_fees_collected, 0),
	COALESCE(setup_completed, FALSE), setup_steps_completed, COALESCE(onboarding_completed, FALSE),
	allowed_roles, restricted_features,
	COALESCE(health_status, 'healthy'), last_health_check, COALESCE(error_count, 0), COALESCE(last_error, ''), last_error_at,
	COALESCE(notes, ''), metadata,
	created_at, updated_at, deleted_at,
	COALESCE(created_by::text, ''), COALESCE(updated_by::text, ''),
	COALESCE(activated_by::text, ''), COALESCE(deactivated_by::text, '')`

// installToolQuery records an installation, on the rules of installModuleQuery.
const installToolQuery = `
	INSERT INTO tenant_tools (
		id, tenant_id, tool_id, module_id, status, is_enabled,
		installed_at, installed_version, integration_enabled, integration_verified,
		health_status, error_count, created_at, updated_at, created_by
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	ON CONFLICT (tenant_id, tool_id) DO UPDATE SET
		module_id = EXCLUDED.module_id, status = EXCLUDED.status, is_enabled = EXCLUDED.is_enabled,
		installed_at = EXCLUDED.installed_at, installed_version = EXCLUDED.installed_version,
		activated_at = NULL, deactivated_at = NULL, activated_by = NULL, deactivated_by = NULL,
		integration_enabled = EXCLUDED.integration_enabled, integration_verified = EXCLUDED.integration_verified,
		integration_verified_at = NULL, integration_status = NULL,
		health_status = EXCLUDED.health_status, error_count = EXCLUDED.error_count,
		last_error = NULL, last_error_at = NULL,
		setup_completed = FALSE, onboarding_completed = FALSE,
		created_by = EXCLUDED.created_by, updated_by = NULL,
		updated_at = EXCLUDED.updated_at, deleted_at = NULL
	WHERE tenant_tools.deleted_at IS NOT NULL
	RETURNING id`

// scanTenantTool decodes one row selected with tenantToolColumns, on the rules of
// scanModule.
func scanTenantTool(row rowScanner) (*registry.TenantTool, error) {
	tt := &registry.TenantTool{}
	var configuration, limits, usage, rateLimits, steps, restricted, metadata []byte

	err := row.Scan(
		&tt.ID, &tt.TenantID, &tt.ToolID, &tt.ModuleID, &tt.Status, &tt.IsEnabled,
		&tt.InstalledAt, &tt.ActivatedAt, &tt.DeactivatedAt, &tt.LastUsedAt,
		&tt.InstalledVersion, &tt.LatestCompatibleVersion,
		&configuration,
		&tt.IntegrationEnabled, &tt.IntegrationStatus, &tt.IntegrationVerified,
		&tt.IntegrationVerifiedAt, &tt.ProviderAccountID,
		&limits, &usage, &rateLimits,
		&tt.SubscriptionStatus, &tt.SubscriptionStart, &tt.SubscriptionEnd, &tt.TrialEndsAt, &tt.NextBillingDate,
		&tt.PricingPlan, &tt.BillingCycle, &tt.AmountPaid, &tt.Currency,
		&tt.TransactionFeesCollected,
		&tt.SetupCompleted, &steps, &tt.OnboardingCompleted,
		pq.Array(&tt.AllowedRoles), &restricted,
		&tt.HealthStatus, &tt.LastHealthCheck, &tt.ErrorCount, &tt.LastError, &tt.LastErrorAt,
		&tt.Notes, &metadata,
		&tt.CreatedAt, &tt.UpdatedAt, &tt.DeletedAt,
		&tt.CreatedBy, &tt.UpdatedBy,
		&tt.ActivatedBy, &tt.DeactivatedBy,
	)
	if err != nil {
		return nil, err
	}

	tt.Configuration, tt.Limits, tt.CurrentUsage, tt.RateLimits = configuration, limits, usage, rateLimits
	tt.SetupStepsCompleted, tt.RestrictedFeatures, tt.Metadata = steps, restricted, metadata
	return tt, nil
}

// Create records an installation, or returns ErrTenantToolAlreadyInstalled when the tenant
// already has this tool. On a reinstall tenantTool.ID becomes the id of the row that was
// brought back.
func (r *tenantToolRepository) Create(ctx context.Context, tenantTool *registry.TenantTool) error {
	return inTenant(ctx, r.db, tenantTool.TenantID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, installToolQuery,
			tenantTool.ID, tenantTool.TenantID, tenantTool.ToolID, nullIfEmpty(tenantTool.ModuleID),
			tenantTool.Status, tenantTool.IsEnabled,
			tenantTool.InstalledAt, tenantTool.InstalledVersion,
			tenantTool.IntegrationEnabled, tenantTool.IntegrationVerified,
			nullIfEmpty(string(tenantTool.HealthStatus)), tenantTool.ErrorCount,
			tenantTool.CreatedAt, tenantTool.UpdatedAt, nullIfEmpty(tenantTool.CreatedBy),
		).Scan(&tenantTool.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return registry.ErrTenantToolAlreadyInstalled
		}
		if err != nil {
			return err
		}
		return recordTx(ctx, tx, r.trail, tenantTool.TenantID, toolKind, tenantTool.ToolID, "installed",
			map[string]any{"version": tenantTool.InstalledVersion})
	})
}

// GetByTenantAndTool retrieves the tenant's live installation of a tool.
func (r *tenantToolRepository) GetByTenantAndTool(ctx context.Context, tenantID, toolID string) (*registry.TenantTool, error) {
	if !isUUID(toolID) {
		return nil, registry.ErrTenantToolNotFound
	}

	var tenantTool *registry.TenantTool
	err := inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		var err error
		tenantTool, err = getTenantTool(ctx, tx, tenantID, toolID, false)
		return err
	})
	return tenantTool, err
}

// getTenantTool reads one live installation inside tx, locking the row when forUpdate is
// set.
func getTenantTool(ctx context.Context, tx *sql.Tx, tenantID, toolID string, forUpdate bool) (*registry.TenantTool, error) {
	query := "SELECT " + tenantToolColumns + ` FROM tenant_tools
		WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL`
	if forUpdate {
		query += " FOR UPDATE"
	}

	tenantTool, err := scanTenantTool(tx.QueryRowContext(ctx, query, tenantID, toolID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, registry.ErrTenantToolNotFound
	}
	return tenantTool, err
}

// CompleteSetup marks an installation's setup done, which also activates one that was
// waiting for it.
func (r *tenantToolRepository) CompleteSetup(ctx context.Context, tenantID, toolID string) error {
	return r.mutate(ctx, tenantID, toolID, "setup_completed", func(tt *registry.TenantTool) error {
		tt.CompleteSetup()
		return nil
	})
}

// updateTenantTool writes the columns the domain methods change, on the rules of
// updateTenantModule. Credentials, configuration, limits, usage and billing are left out.
func updateTenantTool(ctx context.Context, tx *sql.Tx, tt *registry.TenantTool) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE tenant_tools SET
			status = $3, is_enabled = $4,
			activated_at = $5, deactivated_at = $6,
			integration_enabled = $7, integration_status = $8,
			integration_verified = $9, integration_verified_at = $10,
			health_status = $11, last_health_check = $12,
			error_count = $13, last_error = $14, last_error_at = $15,
			setup_completed = $16, onboarding_completed = $17,
			updated_by = $18, activated_by = $19, deactivated_by = $20,
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		tt.ID, tt.TenantID, tt.Status, tt.IsEnabled,
		tt.ActivatedAt, tt.DeactivatedAt,
		tt.IntegrationEnabled, nullIfEmpty(string(tt.IntegrationStatus)),
		tt.IntegrationVerified, tt.IntegrationVerifiedAt,
		nullIfEmpty(string(tt.HealthStatus)), tt.LastHealthCheck,
		tt.ErrorCount, nullIfEmpty(tt.LastError), tt.LastErrorAt,
		tt.SetupCompleted, tt.OnboardingCompleted,
		nullIfEmpty(tt.UpdatedBy), nullIfEmpty(tt.ActivatedBy), nullIfEmpty(tt.DeactivatedBy),
	)
	if err != nil {
		return err
	}
	return requireRow(result, registry.ErrTenantToolNotFound)
}

// Uninstall marks the tenant's installation of a tool deleted and inactive, for the reason
// given on the module repository's Uninstall.
func (r *tenantToolRepository) Uninstall(ctx context.Context, tenantID, toolID string) error {
	if !isUUID(toolID) {
		return registry.ErrTenantToolNotFound
	}

	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE tenant_tools
			SET deleted_at = NOW(), status = 'inactive', is_enabled = FALSE, updated_at = NOW()
			WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL`,
			tenantID, toolID,
		)
		if err != nil {
			return err
		}
		if err := requireRow(result, registry.ErrTenantToolNotFound); err != nil {
			return err
		}
		return recordTx(ctx, tx, r.trail, tenantID, toolKind, toolID, "uninstalled", nil)
	})
}

// ListByTenant retrieves tenant tools with filters
func (r *tenantToolRepository) ListByTenant(ctx context.Context, tenantID string, filters registry.TenantToolFilters) ([]*registry.TenantTool, error) {
	query := "SELECT " + tenantToolColumns + " FROM tenant_tools WHERE tenant_id = $1 AND deleted_at IS NULL"

	conditions, args := r.buildFilterConditions(filters)
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += installationOrderBy(filters.SortBy, filters.SortOrder)
	query += pageClause(filters.Limit, filters.Offset)

	var tenantTools []*registry.TenantTool
	err := inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, append([]any{tenantID}, args...)...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			tt, err := scanTenantTool(rows)
			if err != nil {
				return err
			}
			tenantTools = append(tenantTools, tt)
		}
		return rows.Err()
	})

	return tenantTools, err
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
	if !isUUID(toolID) {
		return false, nil
	}

	var exists bool
	err := inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM tenant_tools
				WHERE tenant_id = $1 AND tool_id = $2
				  AND status = 'active' AND is_enabled = true
				  AND deleted_at IS NULL
			)`, tenantID, toolID,
		).Scan(&exists)
	})
	return exists, err
}

// Activate activates a tool for a tenant
func (r *tenantToolRepository) Activate(ctx context.Context, tenantID, toolID, activatedBy string) error {
	return r.mutate(ctx, tenantID, toolID, "activated", func(tt *registry.TenantTool) error {
		tt.UpdatedBy = activatedBy
		return tt.Activate(activatedBy)
	})
}

// Deactivate deactivates a tool for a tenant
func (r *tenantToolRepository) Deactivate(ctx context.Context, tenantID, toolID, deactivatedBy string) error {
	return r.mutate(ctx, tenantID, toolID, "deactivated", func(tt *registry.TenantTool) error {
		tt.UpdatedBy = deactivatedBy
		return tt.Deactivate(deactivatedBy)
	})
}

// VerifyIntegration marks integration as verified
func (r *tenantToolRepository) VerifyIntegration(ctx context.Context, tenantID, toolID string) error {
	return r.mutate(ctx, tenantID, toolID, "integration_verified", func(tt *registry.TenantTool) error {
		tt.VerifyIntegration()
		return nil
	})
}

// UpdateIntegrationStatus updates the integration status
func (r *tenantToolRepository) UpdateIntegrationStatus(ctx context.Context, tenantID, toolID string, status registry.IntegrationStatus) error {
	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tenant_tools
			SET integration_status = $3, updated_at = NOW()
			WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL`,
			tenantID, toolID, nullIfEmpty(string(status)),
		)
		return err
	})
}

// UpdateHealthStatus updates the health status
func (r *tenantToolRepository) UpdateHealthStatus(ctx context.Context, tenantID, toolID string, status registry.HealthStatus) error {
	return r.mutate(ctx, tenantID, toolID, "", func(tt *registry.TenantTool) error {
		tt.SetHealthStatus(status)
		return nil
	})
}

// RecordError records an error for a tool
func (r *tenantToolRepository) RecordError(ctx context.Context, tenantID, toolID, errorMsg string) error {
	return r.mutate(ctx, tenantID, toolID, "", func(tt *registry.TenantTool) error {
		tt.RecordError(errorMsg)
		return nil
	})
}

// mutate reads an installation under a row lock, applies change, writes it back and records
// action, on the rules of the module repository's mutate.
//
// An empty action records nothing. Health and error counts are the platform observing a
// provider rather than a change somebody made, and a trail that logs every health probe
// buries the changes it exists to show.
func (r *tenantToolRepository) mutate(ctx context.Context, tenantID, toolID, action string, change func(*registry.TenantTool) error) error {
	if !isUUID(toolID) {
		return registry.ErrTenantToolNotFound
	}

	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		tenantTool, err := getTenantTool(ctx, tx, tenantID, toolID, true)
		if err != nil {
			return err
		}
		if err := change(tenantTool); err != nil {
			return err
		}
		if err := updateTenantTool(ctx, tx, tenantTool); err != nil {
			return err
		}
		if action == "" {
			return nil
		}
		return recordTx(ctx, tx, r.trail, tenantID, toolKind, toolID, action,
			map[string]any{"status": string(tenantTool.Status)})
	})
}

// UpdateLastUsed updates the last used timestamp
func (r *tenantToolRepository) UpdateLastUsed(ctx context.Context, tenantID, toolID string) error {
	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tenant_tools
			SET last_used_at = NOW(), updated_at = NOW()
			WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL`,
			tenantID, toolID,
		)
		return err
	})
}

// UpdateUsage updates the current usage for a tool
func (r *tenantToolRepository) UpdateUsage(ctx context.Context, tenantID, toolID string, usage map[string]any) error {
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tenant_tools
			SET current_usage = $3, updated_at = NOW()
			WHERE tenant_id = $1 AND tool_id = $2 AND deleted_at IS NULL`,
			tenantID, toolID, usageJSON,
		)
		return err
	})
}

// Helper functions
func (r *tenantToolRepository) buildFilterConditions(filters registry.TenantToolFilters) ([]string, []any) {
	var conditions []string
	var args []any
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

func tenantToolStatusPtr(s registry.TenantToolStatus) *registry.TenantToolStatus {
	return &s
}
