package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
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

	// Connect to postgres database to create test database
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	defer db.Close()

	// Drop test database if exists
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.DBName))
	require.NoError(t, err)

	// Create test database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
	require.NoError(t, err)

	// Connect to test database
	testConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	testDB, err := sql.Open("postgres", testConnStr)
	require.NoError(t, err)

	// Verify connection
	err = testDB.Ping()
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, testDB)

	// Cleanup function
	t.Cleanup(func() {
		testDB.Close()

		// Drop test database
		db, err := sql.Open("postgres", connStr)
		if err == nil {
			defer db.Close()
			db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.DBName))
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

		ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

		DROP POLICY IF EXISTS tenant_isolation_policy ON tenants;
		CREATE POLICY tenant_isolation_policy ON tenants
			FOR ALL
			USING (id = current_setting('app.current_tenant', TRUE)::UUID);
	`)
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
	require.NoError(t, err)
}

// SetTenantContext sets the tenant context for RLS
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
	_, err := db.ExecContext(ctx, "SET app.current_tenant = $1", tenantID)
	return err
}

// ClearTenantContext clears the tenant context
func ClearTenantContext(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "RESET app.current_tenant")
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
