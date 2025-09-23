package repositories

import (
	"context"
	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/user"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *user.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, tenantID, userID uuid.UUID) (*user.User, error)

	// GetByEmail retrieves a user by email within a tenant
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*user.User, error)

	// GetByEmailAndTenant retrieves a user by email and tenant ID
	GetByEmailAndTenant(ctx context.Context, email string, tenantID shared.TenantID) (*user.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *user.User) error

	// Delete soft deletes a user
	Delete(ctx context.Context, tenantID, userID uuid.UUID) error

	// List retrieves users for a tenant with pagination
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*user.User, error)

	// Count returns the total number of users for a tenant
	Count(ctx context.Context, tenantID uuid.UUID) (int, error)

	// GetByRole retrieves users by role within a tenant
	GetByRole(ctx context.Context, tenantID uuid.UUID, role string) ([]*user.User, error)

	// UpdatePassword updates a user's password
	UpdatePassword(ctx context.Context, tenantID, userID uuid.UUID, hashedPassword string) error

	// ActivateUser activates a user account
	ActivateUser(ctx context.Context, tenantID, userID uuid.UUID) error

	// DeactivateUser deactivates a user account
	DeactivateUser(ctx context.Context, tenantID, userID uuid.UUID) error

	// GetActiveUsers retrieves only active users for a tenant
	GetActiveUsers(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error)

	// Search searches users by name or email within a tenant
	Search(ctx context.Context, tenantID uuid.UUID, query string, limit, offset int) ([]*user.User, error)
}
