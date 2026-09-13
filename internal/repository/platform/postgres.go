// Package platform reads the permission to act across tenants.
//
// It is one table and one question, kept apart from the tenant repositories because it
// answers about the platform rather than about a tenant: an owner in one tenant is nobody
// here until a row says otherwise. See migration 040.
package platform

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository reads platform_operators.
type Repository struct {
	db *sql.DB
}

// New creates the repository.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// IsOperator reports whether the subject may act across tenants.
//
// Complexity: one primary-key lookup, O(log n), on the routes that need it and on no
// other request.
func (r *Repository) IsOperator(ctx context.Context, userID string) (bool, error) {
	var exists bool

	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM platform_operators WHERE user_id = $1)`, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("could not read the platform permission: %w", err)
	}

	return exists, nil
}
