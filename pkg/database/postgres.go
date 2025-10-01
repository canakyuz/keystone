package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"nexspaces-api/internal/config"
)

// DB wraps sql.DB with additional functionality
type DB struct {
	*sql.DB
	logger interface{} // Will be *logger.Logger, kept as interface to avoid circular import
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close gracefully closes the database connection
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// HealthCheck checks if the database is healthy
func HealthCheck(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// SetTenantContext sets the tenant_id in PostgreSQL session for RLS
// This is critical for multi-tenant isolation
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
	query := "SET LOCAL app.current_tenant = $1"
	_, err := db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// GetStats returns database connection pool statistics
func GetStats(db *sql.DB) sql.DBStats {
	return db.Stats()
}

// WithTransaction executes a function within a database transaction
func WithTransaction(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback in case of panic
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	// Execute function
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTenantTransaction executes a function within a transaction with tenant context
func WithTenantTransaction(ctx context.Context, db *sql.DB, tenantID string, fn func(*sql.Tx) error) error {
	return WithTransaction(ctx, db, func(tx *sql.Tx) error {
		// Set tenant context for this transaction
		if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID); err != nil {
			return fmt.Errorf("failed to set tenant context: %w", err)
		}

		// Execute function
		return fn(tx)
	})
}

// CheckConnection verifies database connectivity
func CheckConnection(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("connection check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected result from connection check: got %d, want 1", result)
	}

	return nil
}
