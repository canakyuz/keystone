package database

import (
	"context"
	"database/sql"
	"testing"

	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	pq "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateSchemaName exercises schema name validation. The name is interpolated
// into a SET search_path statement, which cannot be parameterised, so this validation
// is the only thing standing between a caller and SQL injection.
func TestValidateSchemaName(t *testing.T) {
	tests := []struct {
		name        string
		schemaName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid schema name",
			schemaName:  "tenant_acme",
			expectError: false,
		},
		{
			name:        "valid schema containing digits",
			schemaName:  "tenant_acme_123",
			expectError: false,
		},
		{
			name:        "empty schema name",
			schemaName:  "",
			expectError: true,
			errorMsg:    "between 1 and 63 characters",
		},
		{
			name:        "tenant_ prefix olmayan",
			schemaName:  "acme_tenant",
			expectError: true,
			errorMsg:    "tenant_",
		},
		{
			name:        "SQL injection denemesi",
			schemaName:  "tenant_acme; DROP TABLE users;",
			expectError: true,
			errorMsg:    "invalid character",
		},
		{
			name:        "system schema access",
			schemaName:  "pg_catalog",
			expectError: true,
			errorMsg:    "tenant_",
		},
		{
			name:        "containing uppercase",
			schemaName:  "tenant_Acme",
			expectError: true,
			errorMsg:    "invalid character",
		},
		{
			name:        "schema name too long",
			schemaName:  "tenant_" + string(make([]byte, 100)),
			expectError: true,
			errorMsg:    "between 1 and 63 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchemaName(tt.schemaName)

			if tt.expectError {
				assert.Error(t, err, "expected an error but got nil")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Hata beklenmiyordu: %v", err)
			}
		})
	}
}

// TestSetSearchPath exercises setting search_path.
func TestSetSearchPath(t *testing.T) {
	// Build a fake database with sqlmock.
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "could not create sqlmock")
	defer db.Close()

	// The logger may be nil in tests.
	manager := NewTenantConnectionManager(db, nil)

	t.Run("search_path set successfully", func(t *testing.T) {
		schemaName := "tenant_acme"

		// The SQL statement expected to run.
		expectedSQL := `SET search_path TO "tenant_acme", public`

		// Tell the mock this SQL is expected to run.
		mock.ExpectExec(expectedSQL).WillReturnResult(sqlmock.NewResult(0, 0))

		// Test et
		err := manager.SetSearchPath(context.Background(), schemaName)

		// Assertions
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL expectations were not met")
	})

	t.Run("invalid schema name", func(t *testing.T) {
		// A SQL injection attempt.
		schemaName := "tenant_acme; DROP TABLE users;"

		// No mock expectation: validation fails, so no SQL should run at all.

		err := manager.SetSearchPath(context.Background(), schemaName)

		// We expect a validation error.
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("database error", func(t *testing.T) {
		schemaName := "tenant_test"
		expectedSQL := `SET search_path TO "tenant_test", public`

		// The SQL runs but returns an error.
		mock.ExpectExec(expectedSQL).WillReturnError(sql.ErrConnDone)

		err := manager.SetSearchPath(context.Background(), schemaName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not set search_path")
	})
}

// TestResetSearchPath exercises resetting search_path.
func TestResetSearchPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewTenantConnectionManager(db, nil)

	t.Run("reset succeeds", func(t *testing.T) {
		expectedSQL := `SET search_path TO public`
		mock.ExpectExec(expectedSQL).WillReturnResult(sqlmock.NewResult(0, 0))

		err := manager.ResetSearchPath(context.Background())

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestExecuteInTenantContext runs against real PostgreSQL.
//
// WHY not a mock: what this test needs to establish is that the tenant schema is
// REALLY active on the connection the callback receives. sqlmock does not interpret
// the query, it only matches text, so it cannot show whether search_path took effect.
// Indeed, the mocked version of this test passed while the implementation was running
// its queries through the pool.
func TestExecuteInTenantContext(t *testing.T) {
	db := openTestDB(t)
	if db == nil {
		t.Skip("postgres unreachable")
	}
	ctx := context.Background()
	manager := NewTenantConnectionManager(db, nil)

	// The same table name in two schemas, so the active schema can be told apart.
	for schema, value := range map[string]int{"tenant_alpha": 1, "tenant_beta": 2} {
		mustExec(t, db, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pq.QuoteIdentifier(schema)))
		mustExec(t, db, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.marker (id int)", pq.QuoteIdentifier(schema)))
		mustExec(t, db, fmt.Sprintf("TRUNCATE %s.marker", pq.QuoteIdentifier(schema)))
		mustExec(t, db, fmt.Sprintf("INSERT INTO %s.marker VALUES (%d)", pq.QuoteIdentifier(schema), value))
	}

	t.Run("the callback runs in the tenant schema", func(t *testing.T) {
		var got int
		err := manager.ExecuteInTenantContext(ctx, "tenant_alpha", func(conn *sql.Conn) error {
			return conn.QueryRowContext(ctx, "SELECT id FROM marker").Scan(&got)
		})
		require.NoError(t, err)
		assert.Equal(t, 1, got, "the row in the alpha schema should be read")
	})

	t.Run("farkli tenant farkli veri gorur", func(t *testing.T) {
		var got int
		err := manager.ExecuteInTenantContext(ctx, "tenant_beta", func(conn *sql.Conn) error {
			return conn.QueryRowContext(ctx, "SELECT id FROM marker").Scan(&got)
		})
		require.NoError(t, err)
		assert.Equal(t, 2, got, "the row in the beta schema should be read")
	})

	t.Run("a callback error propagates upward", func(t *testing.T) {
		sentinel := errors.New("business logic error")
		err := manager.ExecuteInTenantContext(ctx, "tenant_alpha", func(conn *sql.Conn) error {
			return sentinel
		})
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("gecersiz schema adi reddedilir", func(t *testing.T) {
		called := false
		err := manager.ExecuteInTenantContext(ctx, "public; DROP TABLE users", func(conn *sql.Conn) error {
			called = true
			return nil
		})
		assert.Error(t, err)
		assert.False(t, called, "gecersiz schema'da callback calistirilmamali")
	})

	t.Run("search_path returns to public when the work is done", func(t *testing.T) {
		err := manager.ExecuteInTenantContext(ctx, "tenant_alpha", func(conn *sql.Conn) error {
			return nil
		})
		require.NoError(t, err)

		// Take a fresh connection from the pool and verify it is not dirty.
		var path string
		require.NoError(t, db.QueryRowContext(ctx, "SHOW search_path").Scan(&path))
		assert.NotContains(t, path, "tenant_alpha", "search_path havuza sizdi")
	})
}

func TestGetConnection(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewTenantConnectionManager(db, nil)

	t.Run("connection acquired", func(t *testing.T) {
		conn, err := manager.GetConnection(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, conn)

		// Close the connection, returning it to the pool.
		conn.Close()
	})

	t.Run("cancelled context", func(t *testing.T) {
		// A context that was already cancelled.
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Hemen iptal et

		conn, err := manager.GetConnection(ctx)

		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}

// BenchmarkExecuteInTenantContext measures the cost of entering and leaving a tenant
// context over a real connection pool. What is measured is the connection checkout
// plus the two SET search_path statements.
func BenchmarkExecuteInTenantContext(b *testing.B) {
	db := openBenchDB(b)
	if db == nil {
		b.Skip("postgres unreachable")
	}
	ctx := context.Background()
	manager := NewTenantConnectionManager(db, nil)

	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS tenant_bench`); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ExecuteInTenantContext(ctx, "tenant_bench", func(conn *sql.Conn) error {
			return nil
		})
	}
}
