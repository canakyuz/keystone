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

type tenantModuleRepository struct {
	db    *sql.DB
	trail *auditrepo.Repository
}

// NewTenantModuleRepository creates a new tenant module repository.
//
// Every method runs inside a transaction scoped to the tenant it is given; see inTenant.
// Every change made to an installation goes to the trail in that same transaction, so a
// repository built without one refuses to change anything.
func NewTenantModuleRepository(db *sql.DB, trail *auditrepo.Repository) registry.TenantModuleRepository {
	return &tenantModuleRepository{db: db, trail: trail}
}

// tenantModuleColumns selects an installation in the order scanTenantModule reads it, on
// the rules of moduleColumns.
const tenantModuleColumns = `
	id, tenant_id, module_id, status, is_enabled,
	installed_at, activated_at, deactivated_at, last_used_at,
	installed_version, COALESCE(latest_compatible_version, ''),
	configuration, features_enabled, limits, current_usage,
	COALESCE(subscription_status, ''), subscription_start, subscription_end, trial_ends_at, next_billing_date,
	COALESCE(pricing_plan, ''), COALESCE(billing_cycle, ''), COALESCE(amount_paid, 0), COALESCE(currency, ''),
	COALESCE(setup_completed, FALSE), setup_steps_completed, COALESCE(onboarding_completed, FALSE),
	allowed_roles, restricted_features,
	COALESCE(notes, ''), metadata,
	created_at, updated_at, deleted_at,
	COALESCE(created_by::text, ''), COALESCE(updated_by::text, ''),
	COALESCE(activated_by::text, ''), COALESCE(deactivated_by::text, '')`

// installModuleQuery records an installation.
//
// The table keeps one row per tenant and module, deleted or not, so a plain INSERT would
// refuse every reinstall. A row the tenant uninstalled is brought back with its lifecycle
// reset instead. A row that is still installed is left alone, and then nothing is returned.
const installModuleQuery = `
	INSERT INTO tenant_modules (
		id, tenant_id, module_id, status, is_enabled,
		installed_at, installed_version, created_at, updated_at, created_by
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (tenant_id, module_id) DO UPDATE SET
		status = EXCLUDED.status, is_enabled = EXCLUDED.is_enabled,
		installed_at = EXCLUDED.installed_at, installed_version = EXCLUDED.installed_version,
		activated_at = NULL, deactivated_at = NULL, activated_by = NULL, deactivated_by = NULL,
		setup_completed = FALSE, onboarding_completed = FALSE,
		created_by = EXCLUDED.created_by, updated_by = NULL,
		updated_at = EXCLUDED.updated_at, deleted_at = NULL
	WHERE tenant_modules.deleted_at IS NOT NULL
	RETURNING id`

// scanTenantModule decodes one row selected with tenantModuleColumns, on the rules of
// scanModule.
func scanTenantModule(row rowScanner) (*registry.TenantModule, error) {
	tm := &registry.TenantModule{}
	var configuration, features, limits, usage, steps, restricted, metadata []byte

	err := row.Scan(
		&tm.ID, &tm.TenantID, &tm.ModuleID, &tm.Status, &tm.IsEnabled,
		&tm.InstalledAt, &tm.ActivatedAt, &tm.DeactivatedAt, &tm.LastUsedAt,
		&tm.InstalledVersion, &tm.LatestCompatibleVersion,
		&configuration, &features, &limits, &usage,
		&tm.SubscriptionStatus, &tm.SubscriptionStart, &tm.SubscriptionEnd, &tm.TrialEndsAt, &tm.NextBillingDate,
		&tm.PricingPlan, &tm.BillingCycle, &tm.AmountPaid, &tm.Currency,
		&tm.SetupCompleted, &steps, &tm.OnboardingCompleted,
		pq.Array(&tm.AllowedRoles), &restricted,
		&tm.Notes, &metadata,
		&tm.CreatedAt, &tm.UpdatedAt, &tm.DeletedAt,
		&tm.CreatedBy, &tm.UpdatedBy,
		&tm.ActivatedBy, &tm.DeactivatedBy,
	)
	if err != nil {
		return nil, err
	}

	tm.Configuration, tm.FeaturesEnabled, tm.Limits, tm.CurrentUsage = configuration, features, limits, usage
	tm.SetupStepsCompleted, tm.RestrictedFeatures, tm.Metadata = steps, restricted, metadata
	return tm, nil
}

// Create records an installation, or returns ErrTenantModuleAlreadyInstalled when the
// tenant already has this module. On a reinstall tenantModule.ID becomes the id of the row
// that was brought back.
func (r *tenantModuleRepository) Create(ctx context.Context, tenantModule *registry.TenantModule) error {
	return inTenant(ctx, r.db, tenantModule.TenantID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, installModuleQuery,
			tenantModule.ID, tenantModule.TenantID, tenantModule.ModuleID, tenantModule.Status, tenantModule.IsEnabled,
			tenantModule.InstalledAt, tenantModule.InstalledVersion, tenantModule.CreatedAt, tenantModule.UpdatedAt,
			nullIfEmpty(tenantModule.CreatedBy),
		).Scan(&tenantModule.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return registry.ErrTenantModuleAlreadyInstalled
		}
		if err != nil {
			return err
		}
		return recordTx(ctx, tx, r.trail, tenantModule.TenantID, moduleKind, tenantModule.ModuleID, "installed",
			map[string]any{"version": tenantModule.InstalledVersion})
	})
}

// GetByTenantAndModule retrieves the tenant's live installation of a module.
func (r *tenantModuleRepository) GetByTenantAndModule(ctx context.Context, tenantID, moduleID string) (*registry.TenantModule, error) {
	if !isUUID(moduleID) {
		return nil, registry.ErrTenantModuleNotFound
	}

	var tenantModule *registry.TenantModule
	err := inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		var err error
		tenantModule, err = getTenantModule(ctx, tx, tenantID, moduleID, false)
		return err
	})
	return tenantModule, err
}

// getTenantModule reads one live installation inside tx, locking the row when forUpdate
// is set.
func getTenantModule(ctx context.Context, tx *sql.Tx, tenantID, moduleID string, forUpdate bool) (*registry.TenantModule, error) {
	query := "SELECT " + tenantModuleColumns + ` FROM tenant_modules
		WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL`
	if forUpdate {
		query += " FOR UPDATE"
	}

	tenantModule, err := scanTenantModule(tx.QueryRowContext(ctx, query, tenantID, moduleID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, registry.ErrTenantModuleNotFound
	}
	return tenantModule, err
}

// CompleteSetup marks an installation's setup done, which also activates one that was
// waiting for it.
func (r *tenantModuleRepository) CompleteSetup(ctx context.Context, tenantID, moduleID string) error {
	return r.mutate(ctx, tenantID, moduleID, "setup_completed", func(tm *registry.TenantModule) error {
		tm.CompleteSetup()
		return nil
	})
}

// updateTenantModule writes the lifecycle columns only. Configuration, limits, usage and
// billing are left out: nothing in the API changes them, and writing the whole row back
// would overwrite a concurrent change to them with whatever this request happened to read.
func updateTenantModule(ctx context.Context, tx *sql.Tx, tm *registry.TenantModule) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE tenant_modules SET
			status = $3, is_enabled = $4,
			activated_at = $5, deactivated_at = $6,
			setup_completed = $7, onboarding_completed = $8,
			updated_by = $9, activated_by = $10, deactivated_by = $11,
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		tm.ID, tm.TenantID, tm.Status, tm.IsEnabled,
		tm.ActivatedAt, tm.DeactivatedAt,
		tm.SetupCompleted, tm.OnboardingCompleted,
		nullIfEmpty(tm.UpdatedBy), nullIfEmpty(tm.ActivatedBy), nullIfEmpty(tm.DeactivatedBy),
	)
	if err != nil {
		return err
	}
	return requireRow(result, registry.ErrTenantModuleNotFound)
}

// Uninstall marks the tenant's installation of a module deleted.
//
// It also sets the status to inactive. The install count on modules is kept by a trigger
// that counts active installations and never looks at deleted_at, so a row deleted while
// active would otherwise be counted for good.
func (r *tenantModuleRepository) Uninstall(ctx context.Context, tenantID, moduleID string) error {
	if !isUUID(moduleID) {
		return registry.ErrTenantModuleNotFound
	}

	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE tenant_modules
			SET deleted_at = NOW(), status = 'inactive', is_enabled = FALSE, updated_at = NOW()
			WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL`,
			tenantID, moduleID,
		)
		if err != nil {
			return err
		}
		if err := requireRow(result, registry.ErrTenantModuleNotFound); err != nil {
			return err
		}
		return recordTx(ctx, tx, r.trail, tenantID, moduleKind, moduleID, "uninstalled", nil)
	})
}

// ListByTenant retrieves tenant modules with filters
func (r *tenantModuleRepository) ListByTenant(ctx context.Context, tenantID string, filters registry.TenantModuleFilters) ([]*registry.TenantModule, error) {
	query := "SELECT " + tenantModuleColumns + " FROM tenant_modules WHERE tenant_id = $1 AND deleted_at IS NULL"

	conditions, args := r.buildFilterConditions(filters)
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += installationOrderBy(filters.SortBy, filters.SortOrder)
	query += pageClause(filters.Limit, filters.Offset)

	var tenantModules []*registry.TenantModule
	err := inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, append([]any{tenantID}, args...)...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			tm, err := scanTenantModule(rows)
			if err != nil {
				return err
			}
			tenantModules = append(tenantModules, tm)
		}
		return rows.Err()
	})

	return tenantModules, err
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
	if !isUUID(moduleID) {
		return false, nil
	}

	var exists bool
	err := inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM tenant_modules
				WHERE tenant_id = $1 AND module_id = $2
				  AND status = 'active' AND is_enabled = true
				  AND deleted_at IS NULL
			)`, tenantID, moduleID,
		).Scan(&exists)
	})
	return exists, err
}

// Activate activates a module for a tenant
func (r *tenantModuleRepository) Activate(ctx context.Context, tenantID, moduleID, activatedBy string) error {
	return r.mutate(ctx, tenantID, moduleID, "activated", func(tm *registry.TenantModule) error {
		tm.UpdatedBy = activatedBy
		return tm.Activate(activatedBy)
	})
}

// Deactivate deactivates a module for a tenant
func (r *tenantModuleRepository) Deactivate(ctx context.Context, tenantID, moduleID, deactivatedBy string) error {
	return r.mutate(ctx, tenantID, moduleID, "deactivated", func(tm *registry.TenantModule) error {
		tm.UpdatedBy = deactivatedBy
		return tm.Deactivate(deactivatedBy)
	})
}

// mutate reads an installation under a row lock, applies change, writes it back and records
// action, in one transaction. Two requests changing the same installation take turns,
// instead of the later one writing back a state it read before the earlier one committed.
func (r *tenantModuleRepository) mutate(ctx context.Context, tenantID, moduleID, action string, change func(*registry.TenantModule) error) error {
	if !isUUID(moduleID) {
		return registry.ErrTenantModuleNotFound
	}

	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		tenantModule, err := getTenantModule(ctx, tx, tenantID, moduleID, true)
		if err != nil {
			return err
		}
		if err := change(tenantModule); err != nil {
			return err
		}
		if err := updateTenantModule(ctx, tx, tenantModule); err != nil {
			return err
		}
		return recordTx(ctx, tx, r.trail, tenantID, moduleKind, moduleID, action,
			map[string]any{"status": string(tenantModule.Status)})
	})
}

// UpdateLastUsed updates the last used timestamp
func (r *tenantModuleRepository) UpdateLastUsed(ctx context.Context, tenantID, moduleID string) error {
	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tenant_modules
			SET last_used_at = NOW(), updated_at = NOW()
			WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL`,
			tenantID, moduleID,
		)
		return err
	})
}

// UpdateUsage updates the current usage for a module
func (r *tenantModuleRepository) UpdateUsage(ctx context.Context, tenantID, moduleID string, usage map[string]any) error {
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	return inTenant(ctx, r.db, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tenant_modules
			SET current_usage = $3, updated_at = NOW()
			WHERE tenant_id = $1 AND module_id = $2 AND deleted_at IS NULL`,
			tenantID, moduleID, usageJSON,
		)
		return err
	})
}

// Helper functions
func (r *tenantModuleRepository) buildFilterConditions(filters registry.TenantModuleFilters) ([]string, []any) {
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

func tenantModuleStatusPtr(s registry.TenantModuleStatus) *registry.TenantModuleStatus {
	return &s
}
