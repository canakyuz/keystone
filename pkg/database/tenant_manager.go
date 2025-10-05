package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// TenantManager encapsulates tenant-specific session configuration.
type TenantManager struct {
	db *sql.DB
}

// NewTenantManager creates a new tenant manager for schema-per-tenant isolation.
func NewTenantManager(db *sql.DB) *TenantManager {
	return &TenantManager{db: db}
}

// SetTenantScope sets the PostgreSQL search_path for the current session.
// The setting is connection-scoped; callers should ensure requests reuse the same connection
// (e.g. by executing within a transaction or using middleware that prepares the session).
func (m *TenantManager) SetTenantScope(ctx context.Context, schemaName string) error {
	if schemaName == "" {
		return fmt.Errorf("schema name is required")
	}

	stmt := fmt.Sprintf("SET search_path TO %s, public", pq.QuoteIdentifier(schemaName))
	if _, err := m.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to set tenant search_path: %w", err)
	}

	return nil
}

// ResetTenantScope resets the search_path to default public schema.
func (m *TenantManager) ResetTenantScope(ctx context.Context) error {
	if _, err := m.db.ExecContext(ctx, "RESET search_path"); err != nil {
		return fmt.Errorf("failed to reset tenant search_path: %w", err)
	}
	return nil
}
