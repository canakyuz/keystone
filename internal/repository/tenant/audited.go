package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/internal/domain/tenant"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

// The statements the plain and the audited paths share, so a schema change cannot update
// one and leave the other behind.
const (
	updateTenantQuery = `
		UPDATE tenants SET
			name = $2,
			slug = $3,
			email = $4,
			phone = $5,
			status = $6,
			plan = $7,
			subscription_start = $8,
			subscription_end = $9,
			trial_ends_at = $10,
			custom_domain = $11,
			custom_domain_verified = $12,
			custom_domain_verified_at = $13,
			settings = $14,
			metadata = $15,
			updated_at = $16,
			updated_by = $17
		WHERE id = $1 AND deleted_at IS NULL
	`

	deleteTenantQuery = `
		UPDATE tenants
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	setTenantLocal = `SELECT set_config('app.current_tenant', $1, true)`
)

// ErrNoTrail reports that an audited write was asked of a repository with no trail.
var ErrNoTrail = errors.New("tenant repository: no audit trail configured")

// WithAudit returns a repository that records the trail beside the change.
func (r *PostgresRepository) WithAudit(trail *auditrepo.Repository) *PostgresRepository {
	clone := *r
	clone.trail = trail

	return &clone
}

// UpdateAudited writes the tenant and its audit entry in one transaction.
//
// Suspending a tenant stops every one of its members at their next request. A record of
// that which can go missing while the suspension stands is exactly the record nobody can
// trust afterwards, so the two are one transaction; see rule 7 in docs/INVARIANTS.md.
func (r *PostgresRepository) UpdateAudited(ctx context.Context, t *tenant.Tenant, entry auditrepo.Entry) error {
	settingsJSON, err := json.Marshal(t.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	t.UpdatedAt = time.Now()

	return r.inTenantTx(ctx, t.ID, entry, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, updateTenantQuery,
			t.ID,
			t.Name, t.Slug, t.Email, nullable(t.Phone),
			t.Status, t.Plan,
			t.SubscriptionStart, t.SubscriptionEnd, t.TrialEndsAt,
			nullable(t.CustomDomain), t.CustomDomainVerified, t.CustomDomainVerifiedAt,
			settingsJSON, metadataJSON,
			t.UpdatedAt, nullable(t.UpdatedBy),
		)
		if err != nil {
			return fmt.Errorf("failed to update tenant: %w", err)
		}

		return requireRow(result, tenant.ErrTenantNotFound)
	})
}

// DeleteAudited soft deletes the tenant and records it in the same transaction.
func (r *PostgresRepository) DeleteAudited(ctx context.Context, id string, entry auditrepo.Entry) error {
	return r.inTenantTx(ctx, id, entry, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, deleteTenantQuery, id, time.Now())
		if err != nil {
			return fmt.Errorf("failed to delete tenant: %w", err)
		}

		return requireRow(result, tenant.ErrTenantNotFound)
	})
}

// inTenantTx runs the change and the trail in one transaction.
//
// tenants carries no tenant policy, but audit_log does, so the scope is set for the record
// rather than for the change. It is transaction-local and never reaches a pooled connection.
func (r *PostgresRepository) inTenantTx(
	ctx context.Context, tenantID string, entry auditrepo.Entry, change func(*sql.Tx) error,
) error {
	if r.trail == nil {
		return ErrNoTrail
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin the audited write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, setTenantLocal, tenantID); err != nil {
		return fmt.Errorf("failed to scope the audited write: %w", err)
	}

	if err := change(tx); err != nil {
		return err
	}

	if err := r.trail.AppendTx(ctx, tx, entry); err != nil {
		return err
	}

	return tx.Commit()
}

// requireRow turns "the statement matched nothing" into the domain's own error.
func requireRow(result sql.Result, missing error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return missing
	}

	return nil
}
