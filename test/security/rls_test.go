// Package security verifies that tenant isolation is actually enforced at the
// database layer.
//
// These tests deliberately bypass the repository layer and run SQL directly. The
// point is to exercise the PostgreSQL Row Level Security configuration rather than
// the application logic: even if the application sends a wrong query, the database
// has to refuse a cross-tenant read.
package security

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/test/helpers"
)

// seedUser inserts a user row belonging to the given tenant.
func seedUser(t *testing.T, exec func(string, ...any) error, tenantID, email string) string {
	t.Helper()

	id := uuid.New().String()
	require.NoError(t, exec(`
		INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, role, status)
		VALUES ($1, $2, $3, 'x', 'Test', 'User', 'admin', 'active')`,
		id, tenantID, email))

	return id
}

func TestUsersTable_RejectsCrossTenantReads(t *testing.T) {
	admin := helpers.SetupTestDB(t)
	if admin == nil {
		t.Skip("postgres unreachable")
	}
	ctx := context.Background()

	tenant1 := helpers.CreateTestTenant(t, admin, "iso-tenant-1")
	tenant2 := helpers.CreateTestTenant(t, admin, "iso-tenant-2")

	adminExec := func(q string, args ...any) error {
		_, err := admin.ExecContext(ctx, q, args...)
		return err
	}
	seedUser(t, adminExec, tenant1.ID, "one@example.com")
	seedUser(t, adminExec, tenant2.ID, "two@example.com")

	// RLS is never enforced on a superuser connection. Testing isolation requires a
	// non-superuser role that owns the tables.
	app := helpers.SetupAppRoleDB(t, admin)

	_, err := app.ExecContext(ctx, `SET app.current_tenant = '`+tenant1.ID+`'`)
	require.NoError(t, err)

	t.Run("only the tenant's own rows are visible", func(t *testing.T) {
		var count int
		require.NoError(t, app.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count))
		assert.Equal(t, 1, count, "another tenant's users are visible too")
	})

	t.Run("diger tenant'in kullanicisi ID ile cekilemez", func(t *testing.T) {
		var email string
		err := app.QueryRowContext(ctx,
			`SELECT email FROM users WHERE tenant_id = $1`, tenant2.ID).Scan(&email)
		assert.Error(t, err, "capraz tenant okuma engellenmedi")
	})

	t.Run("no rows are visible without a tenant context", func(t *testing.T) {
		_, err := app.ExecContext(ctx, `RESET app.current_tenant`)
		require.NoError(t, err)

		var count int
		require.NoError(t, app.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count))
		assert.Equal(t, 0, count, "rows visible with no context: fail-open behaviour")
	})
}

// TestAuthLookup_IsTransactionScoped verifies the narrowing introduced by migration
// 028: the global login lookup only sees rows while the flag is set, and the flag
// does not leak outside the transaction.
func TestAuthLookup_IsTransactionScoped(t *testing.T) {
	admin := helpers.SetupTestDB(t)
	if admin == nil {
		t.Skip("postgres unreachable")
	}
	ctx := context.Background()

	tenant1 := helpers.CreateTestTenant(t, admin, "auth-tenant-1")
	adminExec := func(q string, args ...any) error {
		_, err := admin.ExecContext(ctx, q, args...)
		return err
	}
	seedUser(t, adminExec, tenant1.ID, "login@example.com")

	app := helpers.SetupAppRoleDB(t, admin)

	t.Run("the login lookup works while the flag is set", func(t *testing.T) {
		tx, err := app.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		_, err = tx.ExecContext(ctx, `SET LOCAL app.auth_lookup = 'on'`)
		require.NoError(t, err)

		var email string
		require.NoError(t, tx.QueryRowContext(ctx,
			`SELECT email FROM users WHERE email = $1`, "login@example.com").Scan(&email))
		assert.Equal(t, "login@example.com", email)
	})

	t.Run("transaction bitince yetki kapanir", func(t *testing.T) {
		var count int
		require.NoError(t, app.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count))
		assert.Equal(t, 0, count, "auth_lookup bayragi transaction disina sizdi")
	})
}
