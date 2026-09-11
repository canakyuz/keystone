package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	pq "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/pkg/tenantctx"
)

// TestDBConfig holds test database configuration
type TestDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultTestDBConfig returns the test database settings. The values can be
// overridden with environment variables, which is required because the Postgres
// service in CI comes with a different host, port and password.
func DefaultTestDBConfig() *TestDBConfig {
	return &TestDBConfig{
		Host:     envOr("TEST_DB_HOST", "localhost"),
		Port:     envOr("TEST_DB_PORT", "5432"),
		User:     envOr("TEST_DB_USER", "postgres"),
		Password: envOr("TEST_DB_PASSWORD", "postgres"),
		DBName:   envOr("TEST_DB_NAME", "keystone_test"),
		SSLMode:  envOr("TEST_DB_SSLMODE", "disable"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dbCounter produces non-colliding database names for tests in the same process.
var dbCounter atomic.Uint64

// uniqueDBName produces a distinct database name per test.
//
// WHY: every test used to DROP/CREATE one fixed name (keystone_test). Because
// `go test ./...` runs packages in parallel, two tests would try to drop the same
// database at the same time and hit "database is being accessed by other users".
// An isolated name removes the collision by design.
//
// The name is trimmed to Postgres's 63-byte limit.
func uniqueDBName(t *testing.T, base string) string {
	t.Helper()

	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		default:
			return '_'
		}
	}, t.Name())

	name := fmt.Sprintf("%s_%s_%d_%d", base, sanitized, os.Getpid(), dbCounter.Add(1))
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

// dropDatabase drops the database, terminating leftover connections first: the
// sql.DB pool may not release a connection immediately after Close(), and DROP
// DATABASE fails on even a single open connection.
func dropDatabase(db *sql.DB, name string) {
	quoted := pq.QuoteIdentifier(name)

	_, _ = db.Exec(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`, name)

	_, _ = db.Exec("DROP DATABASE IF EXISTS " + quoted)
}

// SetupTestDB opens an isolated per-test database and runs the real migrations. The
// database is dropped when the test finishes. If Postgres is unreachable the test is
// skipped.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := DefaultTestDBConfig()
	dbName := uniqueDBName(t, cfg.DBName)

	adminConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	adminDB := openDBOrSkip(t, adminConnStr)
	if adminDB == nil {
		return nil
	}
	defer adminDB.Close()

	dropDatabase(adminDB, dbName)

	if _, err := adminDB.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName)); skipIfConnectionError(t, err) {
		return nil
	} else {
		require.NoErrorf(t, err, "could not create test database: %s", dbName)
	}

	testConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbName, cfg.SSLMode,
	)

	testDB := openDBOrSkip(t, testConnStr)
	if testDB == nil {
		return nil
	}

	runMigrations(t, testDB)

	t.Cleanup(func() {
		testDB.Close()

		admin := openDBOrSkip(t, adminConnStr)
		if admin != nil {
			defer admin.Close()
			dropDatabase(admin, dbName)
		}
	})

	return testDB
}

// migrationsDir locates the repository's migrations directory relative to this
// file. runtime.Caller is used because `go test` runs each package in its own
// directory, so a path relative to the working directory would differ per package.
func migrationsDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine caller file location")

	// test/helpers/database.go -> repository root
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	dir := filepath.Join(root, "migrations")

	info, err := os.Stat(dir)
	require.NoErrorf(t, err, "migrations directory not found: %s", dir)
	require.Truef(t, info.IsDir(), "%s is not a directory", dir)

	return dir
}

// runMigrations executes the real migrations/*.up.sql files in order.
//
// WHY read from the files: the schema used to be copied by hand into this helper and
// drifted from the production migrations over time. The tenants.phone column, for
// instance, existed in the migration but not in the copy; the tests ran against a
// schema that did not really exist and the repository queries blew up with "column
// does not exist". The single source of truth is the migrations/ directory. Keeping a
// second copy makes that drift inevitable; reading the files makes it structurally
// impossible.
//
// Complexity: O(m) file reads plus O(m) sequential execs, m being the migration count.
func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(migrationsDir(t), "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no migrations/*.up.sql found")

	// Glob does not guarantee order, and 001..032 order is required by schema dependencies.
	sort.Strings(files)

	for _, file := range files {
		stmt, readErr := os.ReadFile(file)
		require.NoErrorf(t, readErr, "could not read migration: %s", file)

		if _, execErr := db.Exec(string(stmt)); skipIfConnectionError(t, execErr) {
			return
		} else {
			require.NoErrorf(t, execErr, "migration failed: %s", filepath.Base(file))
		}
	}
}
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
	if _, err := db.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		return err
	}

	schemaName, err := fetchTenantSchemaName(ctx, db, tenantID)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf("SET search_path TO %s, public", pq.QuoteIdentifier(schemaName))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return err
	}

	return nil
}

// ClearTenantContext clears the tenant context
func ClearTenantContext(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "SELECT set_config('app.current_tenant', '', true)"); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, "RESET search_path")
	return err
}

// WithTenantContext executes a function within a tenant context
func WithTenantContext(ctx context.Context, db *sql.DB, tenantID string, fn func() error) error {
	if err := SetTenantContext(ctx, db, tenantID); err != nil {
		return err
	}
	defer ClearTenantContext(ctx, db)

	return fn()
}

func openDBOrSkip(t *testing.T, connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if skipIfConnectionError(t, err) {
		return nil
	}
	require.NoError(t, err)

	if err := db.Ping(); err != nil {
		if isConnectionError(err) {
			db.Close()
			t.Skipf("Skipping database-backed test: %v", err)
			return nil
		}
		require.NoError(t, err)
	}

	return db
}

func skipIfConnectionError(t *testing.T, err error) bool {
	if err == nil {
		return false
	}

	if isConnectionError(err) {
		t.Skipf("Skipping database-backed test: %v", err)
		return true
	}

	return false
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "connect: operation not permitted") ||
		strings.Contains(msg, "connect: permission denied") ||
		strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "could not connect to server")
}

func fetchTenantSchemaName(ctx context.Context, db *sql.DB, tenantID string) (string, error) {
	var schemaName string
	if err := db.QueryRowContext(ctx, "SELECT schema_name FROM tenants WHERE id = $1", tenantID).Scan(&schemaName); err != nil {
		return "", err
	}
	return schemaName, nil
}

// SetupAppRoleDB returns a connection on which RLS can actually be exercised.
//
// WHY a separate role is needed: in PostgreSQL, superusers bypass Row Level Security
// policies under all circumstances, and FORCE ROW LEVEL SECURITY does not change
// that. Because the tests connect as the "postgres" superuser by default, an
// isolation test run over that connection proves nothing: the policy never engages,
// and the test passes or fails for the wrong reason.
//
// The tables are handed to this role because in production the application usually
// connects as the role that created the schema. Reproducing that ownership scenario
// exactly is what tests whether the FORCE setting from migration 027 actually does
// its job.
//
// Complexity: O(k) ALTERs, k being the number of tables in the public schema.
func SetupAppRoleDB(t *testing.T, admin *sql.DB) *sql.DB {
	t.Helper()

	cfg := DefaultTestDBConfig()

	var dbName string
	require.NoError(t, admin.QueryRow("SELECT current_database()").Scan(&dbName))

	roleName := uniqueDBName(t, "app_role")
	quotedRole := pq.QuoteIdentifier(roleName)

	mustExec := func(query string) {
		_, err := admin.Exec(query)
		require.NoErrorf(t, err, "setup failed: %s", query)
	}

	mustExec("DROP ROLE IF EXISTS " + quotedRole)
	mustExec(fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'approle'", quotedRole))
	mustExec(fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", pq.QuoteIdentifier(dbName), quotedRole))
	mustExec(fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA public TO %s", quotedRole))

	// Hand every public table and sequence to the role.
	mustExec(fmt.Sprintf(`
		DO $$
		DECLARE r record;
		BEGIN
			FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
				EXECUTE format('ALTER TABLE public.%%I OWNER TO %s', r.tablename);
			END LOOP;
			FOR r IN SELECT sequencename FROM pg_sequences WHERE schemaname = 'public' LOOP
				EXECUTE format('ALTER SEQUENCE public.%%I OWNER TO %s', r.sequencename);
			END LOOP;
		END $$`, quotedRole, quotedRole))

	appDB, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=approle dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, roleName, dbName, cfg.SSLMode,
	))
	require.NoError(t, err)

	// SET is per-session. If the pool opens more than one connection, a query can land
	// on a connection that is not carrying the tenant context, which would make the test
	// meaningless.
	appDB.SetMaxOpenConns(1)

	require.NoError(t, appDB.Ping())

	t.Cleanup(func() {
		appDB.Close()
		_, _ = admin.Exec("REASSIGN OWNED BY " + quotedRole + " TO CURRENT_USER")
		_, _ = admin.Exec("DROP OWNED BY " + quotedRole)
		_, _ = admin.Exec("DROP ROLE IF EXISTS " + quotedRole)
	})

	return appDB
}

// WithTenantSchema puts the tenant schema the repositories expect into the context.
//
// WHY: after the TenantConnectionManager refactor the repository methods read the
// schema name from the context and error out when it is absent. In production
// TenantContextMiddleware puts it there; in tests there is no HTTP layer, so the same
// key has to be set directly.
func WithTenantSchema(ctx context.Context, schemaName string) context.Context {
	return tenantctx.WithSchema(ctx, schemaName)
}

// SetupWorkerRoleDB connects as a login role that holds keystone_worker, the group role
// migration 039 defines, and nothing else.
//
// It is neither a superuser nor a table owner, and either would make a worker test pass
// for the wrong reason: a superuser ignores RLS, and an owner holds every privilege on its
// tables whatever the grants say.
func SetupWorkerRoleDB(t *testing.T, admin *sql.DB) *sql.DB {
	t.Helper()

	cfg := DefaultTestDBConfig()

	var dbName string
	require.NoError(t, admin.QueryRow("SELECT current_database()").Scan(&dbName))

	roleName := uniqueDBName(t, "worker_role")
	quotedRole := pq.QuoteIdentifier(roleName)

	for _, stmt := range []string{
		"DROP ROLE IF EXISTS " + quotedRole,
		fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'workerrole' IN ROLE keystone_worker", quotedRole),
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", pq.QuoteIdentifier(dbName), quotedRole),
	} {
		_, err := admin.Exec(stmt)
		require.NoErrorf(t, err, "setup failed: %s", stmt)
	}

	workerDB, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=workerrole dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, roleName, dbName, cfg.SSLMode,
	))
	require.NoError(t, err)
	require.NoError(t, workerDB.Ping())

	t.Cleanup(func() {
		workerDB.Close()
		// The role owns the tenant schemas it created; dropping them goes with it.
		_, _ = admin.Exec("DROP OWNED BY " + quotedRole)
		_, _ = admin.Exec("DROP ROLE IF EXISTS " + quotedRole)
	})

	return workerDB
}
