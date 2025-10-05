package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	pq "github.com/lib/pq"
	"github.com/stretchr/testify/require"
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

// DefaultTestDBConfig returns default test database configuration
func DefaultTestDBConfig() *TestDBConfig {
	return &TestDBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "nexspaces_test",
		SSLMode:  "disable",
	}
}

// SetupTestDB creates a test database connection and runs migrations
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := DefaultTestDBConfig()

	adminConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	adminDB := openDBOrSkip(t, adminConnStr)
	if adminDB == nil {
		return nil
	}
	defer adminDB.Close()

	if _, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.DBName)); skipIfConnectionError(t, err) {
		return nil
	} else {
		require.NoError(t, err)
	}

	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)); skipIfConnectionError(t, err) {
		return nil
	} else {
		require.NoError(t, err)
	}

	testConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
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
			_, _ = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.DBName))
		}
	})

	return testDB
}

// runMigrations runs database migrations for tests
func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	// Migration 001: Create tenants table
	_, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) NOT NULL,
			schema_name VARCHAR(63) UNIQUE NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			plan VARCHAR(20) NOT NULL DEFAULT 'free',
			settings JSONB DEFAULT '{}'::JSONB,
			metadata JSONB DEFAULT '{}'::JSONB,
			custom_domain VARCHAR(255),
			custom_domain_verified BOOLEAN DEFAULT FALSE,
			custom_domain_verified_at TIMESTAMP,
			trial_ends_at TIMESTAMP,
			subscription_ends_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
		CREATE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email);
		CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
		CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_schema_name ON tenants(schema_name);

		ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

		DROP POLICY IF EXISTS tenant_isolation_policy ON tenants;
		CREATE POLICY tenant_isolation_policy ON tenants
			FOR ALL
			USING (id = current_setting('app.current_tenant', TRUE)::UUID);

		CREATE OR REPLACE FUNCTION generate_schema_name(base TEXT)
		RETURNS TEXT AS $$
		DECLARE
			slug TEXT;
			candidate TEXT;
			counter INT := 0;
		BEGIN
			IF base IS NULL OR LENGTH(TRIM(base)) = 0 THEN
				raise exception 'base value cannot be empty';
			END IF;

			slug := lower(regexp_replace(base, '[^a-zA-Z0-9]+', '_', 'g'));
			slug := regexp_replace(slug, '_+', '_', 'g');
			slug := trim(both '_' FROM slug);
			IF slug = '' THEN
				slug := 'tenant';
			END IF;

			candidate := 'tenant_' || slug;

			WHILE EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = candidate)
				OR EXISTS (SELECT 1 FROM tenants WHERE schema_name = candidate) LOOP
				counter := counter + 1;
				candidate := 'tenant_' || slug || '_' || counter;
			END LOOP;

			RETURN candidate;
		END;
		$$ LANGUAGE plpgsql;
	`)
	if skipIfConnectionError(t, err) {
		return
	}
	require.NoError(t, err)

	// Migration 002: Create users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			email VARCHAR(255) NOT NULL,
			password VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			email_verified BOOLEAN DEFAULT FALSE,
			email_verified_at TIMESTAMP,
			last_login_at TIMESTAMP,
			avatar VARCHAR(500),
			phone VARCHAR(50),
			timezone VARCHAR(50),
			locale VARCHAR(10),
			two_factor_enabled BOOLEAN DEFAULT FALSE,
			password_changed_at TIMESTAMP,
			preferences JSONB DEFAULT '{}'::JSONB,
			metadata JSONB DEFAULT '{}'::JSONB,
			created_by UUID,
			updated_by UUID,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP,
			UNIQUE(tenant_id, email)
		);

		CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email);
		CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
		CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

		ALTER TABLE users ENABLE ROW LEVEL SECURITY;

		DROP POLICY IF EXISTS tenant_isolation_policy ON users;
		CREATE POLICY tenant_isolation_policy ON users
			FOR ALL
			USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
	`)
	if skipIfConnectionError(t, err) {
		return
	}
	require.NoError(t, err)

	// Migration 003: Create websites table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS websites (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(50) NOT NULL,
			description TEXT,
			template_id VARCHAR(50) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'draft',
			domain VARCHAR(255),
			settings JSONB DEFAULT '{}'::JSONB,
			content JSONB DEFAULT '{}'::JSONB,
			metadata JSONB DEFAULT '{}'::JSONB,
			published_at TIMESTAMP,
			archived_at TIMESTAMP,
			created_by UUID,
			updated_by UUID,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP,
			UNIQUE(tenant_id, slug)
		);

		CREATE INDEX IF NOT EXISTS idx_websites_tenant_id ON websites(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_websites_slug ON websites(slug);
		CREATE INDEX IF NOT EXISTS idx_websites_tenant_slug ON websites(tenant_id, slug);
		CREATE INDEX IF NOT EXISTS idx_websites_status ON websites(status);

		ALTER TABLE websites ENABLE ROW LEVEL SECURITY;

		DROP POLICY IF EXISTS tenant_isolation_policy ON websites;
		CREATE POLICY tenant_isolation_policy ON websites
			FOR ALL
			USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

		DROP POLICY IF EXISTS public_websites_policy ON websites;
		CREATE POLICY public_websites_policy ON websites
			FOR SELECT
			USING (status = 'published' AND deleted_at IS NULL);
	`)
	if skipIfConnectionError(t, err) {
		return
	}
	require.NoError(t, err)
}

// SetTenantContext sets the tenant context for RLS
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
