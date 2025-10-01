package user

import (
	"context"

	"nexpaces-api/internal/domain/user"
)

// Repository defines the interface for user data operations
type Repository interface {
	// Create creates a new user
	Create(ctx context.Context, u *user.User) error

	// GetByID retrieves a user by ID (tenant-scoped)
	GetByID(ctx context.Context, tenantID, userID string) (*user.User, error)

	// GetByEmail retrieves a user by email (tenant-scoped)
	GetByEmail(ctx context.Context, tenantID, email string) (*user.User, error)

	// GetByEmailGlobal retrieves a user by email across all tenants (for auth)
	GetByEmailGlobal(ctx context.Context, email string) (*user.User, error)

	// List retrieves all users in a tenant with pagination
	List(ctx context.Context, tenantID string, filters ListFilters) ([]*user.User, int64, error)

	// Update updates an existing user
	Update(ctx context.Context, u *user.User) error

	// Delete soft deletes a user
	Delete(ctx context.Context, tenantID, userID string) error

	// ExistsByEmail checks if a user with the given email exists in tenant
	ExistsByEmail(ctx context.Context, tenantID, email string) (bool, error)

	// CountByTenant counts users in a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// CountByRole counts users by role in a tenant
	CountByRole(ctx context.Context, tenantID string, role user.UserRole) (int64, error)

	// CountByStatus counts users by status in a tenant
	CountByStatus(ctx context.Context, tenantID string, status user.UserStatus) (int64, error)

	// UpdateLastLogin updates user's last login timestamp
	UpdateLastLogin(ctx context.Context, tenantID, userID string) error

	// GetOwner retrieves the tenant owner
	GetOwner(ctx context.Context, tenantID string) (*user.User, error)
}

// ListFilters represents filters for listing users
type ListFilters struct {
	// Pagination
	Limit  int
	Offset int

	// Filters
	Role   *user.UserRole
	Status *user.UserStatus
	Search string // Search in email, first_name, last_name

	// Sorting
	SortBy    string // created_at, email, first_name, last_name
	SortOrder string // asc, desc
}
