package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/canakyuz/keystone/internal/domain/registry"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

// ErrNoTrail reports that a change to an installation was asked of a repository built
// without an audit trail.
var ErrNoTrail = errors.New("registry repository: no audit trail configured")

// installationKind names what an installation installs: the subject type its audit entries
// carry and the catalogue table its id points into.
type installationKind struct {
	subject   string
	catalogue string
}

var (
	moduleKind = installationKind{subject: "module", catalogue: "modules"}
	toolKind   = installationKind{subject: "tool", catalogue: "tools"}
)

// setTenantLocal scopes the rest of a transaction to one tenant.
//
// tenant_modules and tenant_tools are under FORCE ROW LEVEL SECURITY with a fail-closed
// policy: a statement that runs without this setting sees no row and may write none.
const setTenantLocal = `SELECT set_config('app.current_tenant', $1, true)`

// inTenant runs fn in a transaction scoped to tenantID and commits it when fn succeeds.
//
// The setting is transaction-local, so it ends with the transaction and cannot travel on
// the pooled connection into another tenant's request.
func inTenant(ctx context.Context, db *sql.DB, tenantID string, fn func(*sql.Tx) error) error {
	if !isUUID(tenantID) {
		return registry.ErrTenantIDRequired
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin the tenant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, setTenantLocal, tenantID); err != nil {
		return fmt.Errorf("failed to scope the transaction to the tenant: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// recordTx appends the audit entry for a change to an installation, inside the transaction
// that makes the change; see rule 7 in docs/INVARIANTS.md.
//
// The actor is read from the request context rather than passed in, so a call site cannot
// record the wrong one by forgetting. The catalogue entry's code and name go into the
// metadata, so the line stays legible after the catalogue renames or removes the entry.
func recordTx(
	ctx context.Context, tx *sql.Tx, trail *auditrepo.Repository,
	tenantID string, kind installationKind, subjectID, action string, metadata map[string]any,
) error {
	if trail == nil {
		return ErrNoTrail
	}

	var code, name string
	err := tx.QueryRowContext(ctx, "SELECT code, name FROM "+kind.catalogue+" WHERE id = $1", subjectID).Scan(&code, &name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to name the audited %s: %w", kind.subject, err)
	}

	details := map[string]any{"code": code, "name": name}
	for key, value := range metadata {
		details[key] = value
	}

	actorID, actorType := auditrepo.ActorFrom(ctx)
	return trail.AppendTx(ctx, tx, auditrepo.Entry{
		TenantID:    tenantID,
		ActorID:     actorID,
		ActorType:   actorType,
		Action:      kind.subject + "." + action,
		SubjectType: kind.subject,
		SubjectID:   subjectID,
		Metadata:    details,
	})
}

// nullIfEmpty sends an empty string as NULL. The optional text columns of the installation
// tables carry CHECK constraints that accept NULL and refuse an empty string, and the
// optional UUID columns refuse one outright.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// requireRow turns "the statement matched nothing" into the given domain error.
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

// installationSortColumns maps the sort keys an installation list accepts to their
// columns. The key is spliced into ORDER BY, so only a name found here reaches the SQL; see
// catalogSortColumns.
var installationSortColumns = map[string]string{
	"installed_at": "installed_at",
	"activated_at": "activated_at",
	"last_used_at": "last_used_at",
}

// installationOrderBy builds the ORDER BY clause for an installation list. Timestamps that
// were never set sort last, and the id keeps equal rows in one order across pages.
func installationOrderBy(sortBy, sortOrder string) string {
	column, ok := installationSortColumns[sortBy]
	if !ok {
		column = "installed_at"
	}

	direction := "DESC"
	if sortOrder == "asc" {
		direction = "ASC"
	}

	return " ORDER BY " + column + " " + direction + " NULLS LAST, id"
}

// pageClause renders LIMIT and OFFSET from already-validated integers.
func pageClause(limit, offset int) string {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}
