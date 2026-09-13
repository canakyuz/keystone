package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/internal/domain/user"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

// The statements the plain and the audited paths share.
//
// One statement, two callers: the audited path runs inside a transaction it also writes the
// trail in, and the plain path runs on the tenant-pinned connection. Copying the SQL would
// mean a schema change that updates one of them and leaves the other behind.
const (
	updateUserQuery = `
		UPDATE users SET
			email = $3,
			password_hash = $4,
			first_name = $5,
			last_name = $6,
			role = $7,
			status = $8,
			email_verified = $9,
			email_verified_at = $10,
			last_login_at = $11,
			avatar = $12,
			phone = $13,
			timezone = $14,
			locale = $15,
			two_factor_enabled = $16,
			password_changed_at = $17,
			preferences = $18,
			metadata = $19,
			updated_at = $20,
			updated_by = $21
		WHERE id = $1 AND tenant_id = $2
	`

	deleteUserQuery = `
		UPDATE users
		SET deleted_at = NOW(), status = 'inactive'
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	setTenantLocal = `SELECT set_config('app.current_tenant', $1, true)`
)

// ErrNoTrail reports that an audited write was asked of a repository with no trail.
var ErrNoTrail = errors.New("user repository: no audit trail configured")

// WithAudit returns a repository that records the trail beside the change.
//
// A separate constructor rather than a required parameter, so the many read paths that do
// not change anything are not forced to supply one.
func (r *PostgresRepository) WithAudit(trail *auditrepo.Repository) *PostgresRepository {
	clone := *r
	clone.trail = trail

	return &clone
}

// UpdateAudited writes the user and its audit entry in one transaction.
//
// The two belong together for the reason rule 7 in docs/INVARIANTS.md gives: a record that
// can disagree with the row it describes is worse than no record. Written after the change,
// a crash in between leaves a suspension nobody can account for; written before, a rollback
// leaves a record of something that never happened.
func (r *PostgresRepository) UpdateAudited(ctx context.Context, u *user.User, entry auditrepo.Entry) error {
	preferencesJSON, err := json.Marshal(u.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	metadataJSON, err := json.Marshal(u.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	u.UpdatedAt = time.Now()

	return r.inTenantTx(ctx, u.TenantID, entry, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, updateUserQuery,
			u.ID, u.TenantID,
			u.Email, u.Password, u.FirstName, u.LastName, u.Role, u.Status,
			u.EmailVerified, u.EmailVerifiedAt, u.LastLoginAt,
			nullable(u.Avatar), nullable(u.Phone), nullable(u.Timezone), nullable(u.Locale),
			u.TwoFactorEnabled, u.PasswordChangedAt,
			preferencesJSON, metadataJSON,
			u.UpdatedAt, nullable(u.UpdatedBy),
		)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		return requireRow(result, user.ErrUserNotFound)
	})
}

// DeleteAudited soft deletes the user and records it in the same transaction.
func (r *PostgresRepository) DeleteAudited(ctx context.Context, tenantID, userID string, entry auditrepo.Entry) error {
	return r.inTenantTx(ctx, tenantID, entry, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, deleteUserQuery, userID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return requireRow(result, user.ErrUserNotFound)
	})
}

// inTenantTx runs the change and the trail in one transaction, scoped to the tenant.
//
// The scope is set with set_config(..., true), so it lasts as long as the transaction and
// never reaches a pooled connection. Both tables it writes carry the fail-closed tenant
// policy, so without it the change and the record would both be refused.
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
