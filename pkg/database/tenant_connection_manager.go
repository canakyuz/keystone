package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/lib/pq"

	"github.com/canakyuz/keystone/pkg/tenantctx"

	"github.com/canakyuz/keystone/pkg/logger"
)

// TenantConnectionManager provides tenant isolation in a schema-per-tenant setup.
//
// In PostgreSQL, search_path decides which schema a query runs against. This
// manager guarantees that the right tenant schema is set for every request and
// cleared again afterwards.
type TenantConnectionManager interface {
	// SetSearchPath activates the given tenant schema.
	// WARNING: it must be cleared with ResetSearchPath; use defer.
	SetSearchPath(ctx context.Context, schemaName string) error

	// ResetSearchPath returns search_path to its default (public). This is a
	// security-critical step: it is what stops cross-tenant data leaking through a
	// pooled connection.
	ResetSearchPath(ctx context.Context) error

	// ExecuteInTenantContext runs a function inside the tenant schema context.
	//
	// The callback receives the CONNECTION whose search_path was set, and MUST run
	// every one of its queries on that connection. Running a query through the pool
	// (*sql.DB) silently lands on a different connection, and that connection's
	// search_path is not set to this tenant.
	//
	// Example:
	//   err := manager.ExecuteInTenantContext(ctx, "tenant_acme", func(conn *sql.Conn) error {
	//       return conn.QueryRowContext(ctx, "SELECT ...").Scan(&x)
	//   })
	ExecuteInTenantContext(ctx context.Context, schemaName string, fn func(conn *sql.Conn) error) error

	// GetConnection takes a fresh connection from the pool. For advanced cases;
	// prefer ExecuteInTenantContext.
	GetConnection(ctx context.Context) (*sql.Conn, error)
}

// setCurrentTenant sets the session variable the RLS policies read.
//
// set_config with is_local=false is used rather than SET LOCAL, because the work runs
// on a pinned connection outside any transaction. SET LOCAL would be discarded
// immediately and the policies would still see nothing.
const setCurrentTenant = `SELECT set_config('app.current_tenant', $1, false)`

// connectionManager is the TenantConnectionManager implementation.
type connectionManager struct {
	db     *sql.DB
	logger *logger.Logger
	mu     sync.RWMutex
}

// NewTenantConnectionManager creates a new TenantConnectionManager.
func NewTenantConnectionManager(db *sql.DB, log *logger.Logger) TenantConnectionManager {
	return &connectionManager{
		db:     db,
		logger: log,
	}
}

// SetSearchPath changes the search_path of the database connection.
func (m *connectionManager) SetSearchPath(ctx context.Context, schemaName string) error {
	if err := validateSchemaName(schemaName); err != nil {
		return fmt.Errorf("invalid schema name: %w", err)
	}

	// pq.QuoteIdentifier guards against SQL injection here.
	quotedSchema := pq.QuoteIdentifier(schemaName)
	setCmd := fmt.Sprintf("SET search_path TO %s, public", quotedSchema)

	if _, err := m.db.ExecContext(ctx, setCmd); err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"schema_name": schemaName,
				"error":       err.Error(),
			}).Error("could not set search_path")
		}
		return fmt.Errorf("could not set search_path: %w", err)
	}

	if m.logger != nil {
		m.logger.WithFields(logger.Fields{
			"schema_name": schemaName,
		}).Debug("search_path set")
	}

	return nil
}

// ResetSearchPath returns search_path to the public schema.
func (m *connectionManager) ResetSearchPath(ctx context.Context) error {
	resetCmd := "SET search_path TO public"

	if _, err := m.db.ExecContext(ctx, resetCmd); err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"error": err.Error(),
			}).Error("could not reset search_path")
		}
		return fmt.Errorf("could not reset search_path: %w", err)
	}

	if m.logger != nil {
		m.logger.Debug("search_path reset to public")
	}

	return nil
}

// ExecuteInTenantContext runs the given function in the tenant schema context.
//
// Why sql.Conn rather than sql.DB:
//   - sql.DB is the pool, that is, several connections.
//   - sql.Conn is ONE connection checked out of that pool.
//   - search_path is set on the sql.Conn, so other requests are unaffected.
//   - Close() returns the connection to the pool for reuse.
//
// ExecuteInTenantContext checks out a single connection, activates the tenant
// schema on it, and hands THAT SAME connection to the callback.
//
// WHY the callback receives the connection: the previous signature was
// fn func() error. A connection was checked out and search_path was written to it,
// but since the callback could not reach it, the queries ran through the pool
// (*sql.DB) instead. The bug cut both ways: the tenant schema was never actually
// active for the queries, and under concurrent load a query could land on a
// connection still set to another tenant and not yet reset. Moving the connection
// into the signature closes this whole class at compile time.
//
// Complexity: O(1) to check out the connection, plus whatever the callback costs.
func (m *connectionManager) ExecuteInTenantContext(ctx context.Context, schemaName string, fn func(conn *sql.Conn) error) error {
	if err := validateSchemaName(schemaName); err != nil {
		return fmt.Errorf("invalid schema name: %w", err)
	}

	// Take a dedicated connection from the pool.
	conn, err := m.db.Conn(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"error": err.Error(),
			}).Error("could not acquire database connection")
		}
		return fmt.Errorf("could not acquire database connection: %w", err)
	}

	// Return the connection to the pool, even on panic.
	defer conn.Close()

	// Activate the tenant schema.
	quotedSchema := pq.QuoteIdentifier(schemaName)
	setCmd := fmt.Sprintf("SET search_path TO %s, public", quotedSchema)

	if _, err := conn.ExecContext(ctx, setCmd); err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"schema_name": schemaName,
				"error":       err.Error(),
			}).Error("could not set search_path on connection")
		}
		return fmt.Errorf("could not set search_path: %w", err)
	}

	// Set the RLS session variable on the SAME connection.
	//
	// WHY this is here and not optional: search_path only decides which schema an
	// unqualified name resolves to. It has no effect on Row Level Security. The
	// policies on the shared tables read app.current_tenant, so without this the
	// application sees the empty set for every RLS-protected table.
	//
	// This was invisible for a long time because the repository tests connect as the
	// superuser, and superusers bypass RLS. Under the non-superuser role SECURITY.md
	// requires in production, every such read returned nothing. test/e2e is what
	// surfaced it.
	//
	// The value is bound as a parameter, never interpolated.
	if tenantID := tenantctx.ID(ctx); tenantID != "" {
		if _, err := conn.ExecContext(ctx, setCurrentTenant, tenantID); err != nil {
			if m.logger != nil {
				m.logger.WithFields(logger.Fields{
					"tenant_id": tenantID,
					"error":     err.Error(),
				}).Error("could not set app.current_tenant on connection")
			}
			return fmt.Errorf("could not set tenant context: %w", err)
		}
	}

	// Clear search_path when the work is done. This one is security-critical.
	defer func() {
		// The reset has to happen even if the context was cancelled.
		resetCtx := context.Background()

		if _, err := conn.ExecContext(resetCtx, "SET search_path TO public"); err != nil {
			if m.logger != nil {
				m.logger.WithFields(logger.Fields{
					"schema_name": schemaName,
					"error":       err.Error(),
				}).Error("could not reset search_path in defer")
			}
		}

		// Clear the tenant as well. A connection going back to the pool still carrying
		// app.current_tenant would let the next request read the previous tenant's rows
		// if it forgot to set its own.
		if _, err := conn.ExecContext(resetCtx, setCurrentTenant, ""); err != nil {
			if m.logger != nil {
				m.logger.WithFields(logger.Fields{
					"error": err.Error(),
				}).Error("could not reset app.current_tenant in defer")
			}
		}
	}()

	// Run the caller's function.
	if m.logger != nil {
		m.logger.WithFields(logger.Fields{
			"schema_name": schemaName,
		}).Debug("running function in tenant context")
	}

	return fn(conn)
}

// GetConnection returns a fresh connection from the pool.
func (m *connectionManager) GetConnection(ctx context.Context) (*sql.Conn, error) {
	conn, err := m.db.Conn(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.WithFields(logger.Fields{
				"error": err.Error(),
			}).Error("could not acquire database connection")
		}
		return nil, fmt.Errorf("could not acquire database connection: %w", err)
	}

	return conn, nil
}

// validateSchemaName verifies that a schema name is safe to interpolate.
//
// Kurallar:
//   - must start with 'tenant_'
//   - lowercase letters, digits and underscore only
//
// - Maksimum 63 karakter (PostgreSQL limiti)
//
// Reason: the schema name in a SET search_path statement cannot be parameterised,
// so validating it by hand is the only defence against SQL injection.
func validateSchemaName(schemaName string) error {
	if len(schemaName) == 0 || len(schemaName) > 63 {
		return fmt.Errorf("schema name must be between 1 and 63 characters")
	}

	// Every tenant schema starts with 'tenant_'. That prefix is what keeps the system
	// schemas (pg_catalog, information_schema) out of reach.
	if len(schemaName) < 7 || schemaName[:7] != "tenant_" {
		return fmt.Errorf("schema name must start with 'tenant_'")
	}

	// Allow safe characters only.
	for i, ch := range schemaName {
		isLower := ch >= 'a' && ch <= 'z'
		isDigit := ch >= '0' && ch <= '9'
		isUnderscore := ch == '_'

		if !isLower && !isDigit && !isUnderscore {
			return fmt.Errorf("invalid character in schema name at position %d: %c", i, ch)
		}
	}

	return nil
}
