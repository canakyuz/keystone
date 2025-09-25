package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/user"
	"nexspaces-api/internal/core/ports/repositories"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UserRepository implements the user repository interface for PostgreSQL
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *sql.DB) repositories.UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, user *user.User) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(user.TenantID)); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		INSERT INTO users (
			id, tenant_id, email, first_name, last_name, password_hash,
			role, status, email_verified, last_login_at, settings, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	var lastLoginAt *sql.NullTime
	if user.LastLoginAt != nil {
		lastLoginAt = &sql.NullTime{
			Time:  *user.LastLoginAt,
			Valid: true,
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.TenantID,
		user.Email,
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		string(user.Role),
		string(user.Status),
		user.EmailVerified,
		lastLoginAt,
		user.Settings,
		user.Metadata,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "users_tenant_id_email_key" {
					return shared.ErrUserAlreadyExists
				}
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, tenantID, userID uuid.UUID) (*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND id = $2`

	row := r.db.QueryRowContext(ctx, query, tenantID, userID)

	return r.scanUser(row)
}

// GetByEmail retrieves a user by email within a tenant
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND email = $2`

	row := r.db.QueryRowContext(ctx, query, tenantID, email)

	return r.scanUser(row)
}

// GetByEmailAndTenant retrieves a user by email within a specific tenant
func (r *UserRepository) GetByEmailAndTenant(ctx context.Context, email string, tenantID shared.TenantID) (*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE email = $1 AND tenant_id = $2`

	row := r.db.QueryRowContext(ctx, query, email, uuid.UUID(tenantID))

	return r.scanUser(row)
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *user.User) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(user.TenantID)); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		UPDATE users
		SET email = $2, first_name = $3, last_name = $4, password_hash = $5,
			role = $6, status = $7, email_verified = $8, last_login_at = $9,
			settings = $10, metadata = $11, updated_at = $12
		WHERE id = $1`

	var lastLoginAt *sql.NullTime
	if user.LastLoginAt != nil {
		lastLoginAt = &sql.NullTime{
			Time:  *user.LastLoginAt,
			Valid: true,
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		string(user.Role),
		string(user.Status),
		user.EmailVerified,
		lastLoginAt,
		user.Settings,
		user.Metadata,
		user.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "users_tenant_id_email_key" {
					return shared.ErrUserAlreadyExists
				}
			}
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrUserNotFound
	}

	return nil
}

// Delete soft deletes a user (sets status to inactive)
func (r *UserRepository) Delete(ctx context.Context, tenantID, userID uuid.UUID) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		UPDATE users
		SET status = 'inactive', updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2 AND status != 'inactive'`

	result, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrUserNotFound
	}

	return nil
}

// ListByTenant retrieves users within a tenant with pagination and filtering
func (r *UserRepository) ListByTenant(ctx context.Context, tenantID shared.TenantID, criteria repositories.UserListCriteria) ([]*user.User, int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, 0, fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Build WHERE clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{uuid.UUID(tenantID)}
	argCount := 1

	if criteria.Status != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, string(*criteria.Status))
	}

	if criteria.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+criteria.Search+"%")
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM users " + whereClause
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Build main query with pagination
	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`

	argCount++
	query = fmt.Sprintf(query, argCount, argCount+1)
	args = append(args, criteria.PageSize, (criteria.Page-1)*criteria.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return users, totalCount, nil
}

// Count returns the total number of users for a tenant
func (r *UserRepository) Count(ctx context.Context, tenantID uuid.UUID) (int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return 0, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := "SELECT COUNT(*) FROM users WHERE tenant_id = $1"

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by tenant: %w", err)
	}

	return count, nil
}

// CountByTenant counts users in a tenant
func (r *UserRepository) CountByTenant(ctx context.Context, tenantID shared.TenantID) (int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return 0, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND status = 'active'"

	var count int
	err := r.db.QueryRowContext(ctx, query, uuid.UUID(tenantID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by tenant: %w", err)
	}

	return count, nil
}

// GetTenantAdmins retrieves all admin users for a tenant
func (r *UserRepository) GetTenantAdmins(ctx context.Context, tenantID shared.TenantID) ([]*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND role IN ('owner', 'admin') AND status = 'active'
		ORDER BY role, created_at`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant admins: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id shared.UserID, timestamp shared.Timestamp) error {
	query := `
		UPDATE users
		SET last_login_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, uuid.UUID(id), timestamp.Time())
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrUserNotFound
	}

	return nil
}

// ActivateUser activates a user account
func (r *UserRepository) ActivateUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	query := `
		UPDATE users
		SET status = 'active', updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2 AND status != 'active'`

	result, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrUserNotFound
	}

	return nil
}

// DeactivateUser deactivates a user account
func (r *UserRepository) DeactivateUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	query := `
		UPDATE users
		SET status = 'inactive', updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2 AND status != 'inactive'`

	result, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrUserNotFound
	}

	return nil
}

// GetActiveUsers retrieves only active users for a tenant
func (r *UserRepository) GetActiveUsers(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// GetByRole retrieves users by role within a tenant
func (r *UserRepository) GetByRole(ctx context.Context, tenantID uuid.UUID, role string) ([]*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND role = $2
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, tenantID, userID uuid.UUID, hashedPassword string) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		UPDATE users
		SET password_hash = $3, updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, userID, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrUserNotFound
	}

	return nil
}

// Search searches users by name or email within a tenant
func (r *UserRepository) Search(ctx context.Context, tenantID uuid.UUID, searchQuery string, limit, offset int) ([]*user.User, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, shared.TenantID(tenantID)); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, first_name, last_name, password_hash,
			   role, status, email_verified, last_login_at, settings, metadata,
			   created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND (first_name ILIKE $2 OR last_name ILIKE $2 OR email ILIKE $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, tenantID, "%"+searchQuery+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// List retrieves users for a tenant with pagination
func (r *UserRepository) List(ctx context.Context, criteria repositories.UserListCriteria) ([]*user.User, int, error) {
	return r.ListByTenant(ctx, shared.TenantID(criteria.TenantID), criteria)
}

// setTenantContext sets the tenant context for Row Level Security
func (r *UserRepository) setTenantContext(ctx context.Context, tenantID shared.TenantID) error {
	query := "SELECT set_tenant_context($1)"
	_, err := r.db.ExecContext(ctx, query, uuid.UUID(tenantID))
	return err
}

// scanUser is a helper function to scan user from database row
func (r *UserRepository) scanUser(scanner interface {
	Scan(dest ...interface{}) error
}) (*user.User, error) {
	var (
		id            uuid.UUID
		tenantID      uuid.UUID
		email         string
		firstName     string
		lastName      string
		passwordHash  sql.NullString
		role          string
		status        string
		emailVerified bool
		lastLoginAt   sql.NullTime
		settings      []byte
		metadata      []byte
		createdAt     sql.NullTime
		updatedAt     sql.NullTime
	)

	err := scanner.Scan(
		&id, &tenantID, &email, &firstName, &lastName, &passwordHash,
		&role, &status, &emailVerified, &lastLoginAt, &settings, &metadata,
		&createdAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	userRole, err := user.ParseRole(role)
	if err != nil {
		return nil, fmt.Errorf("invalid user role in database: %w", err)
	}

	userStatus, err := user.ParseStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid user status in database: %w", err)
	}

	var lastLogin *time.Time
	if lastLoginAt.Valid {
		lastLogin = &lastLoginAt.Time
	}

	var passwordHashPtr *string
	if passwordHash.Valid {
		passwordHashPtr = &passwordHash.String
	}

	u := &user.User{
		ID:            id,
		TenantID:      tenantID,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		PasswordHash:  passwordHashPtr,
		Role:          userRole,
		Status:        userStatus,
		EmailVerified: emailVerified,
		LastLoginAt:   lastLogin,
		Settings:      make(map[string]interface{}),
		Metadata:      make(map[string]interface{}),
		CreatedAt:     createdAt.Time,
		UpdatedAt:     updatedAt.Time,
	}

	return u, nil
}
