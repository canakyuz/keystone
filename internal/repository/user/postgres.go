package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nexpaces-api/internal/domain/user"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new user
func (r *PostgresRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (
			id, tenant_id, email, password_hash, first_name, last_name, role, status,
			email_verified, email_verified_at, last_login_at,
			avatar, phone, timezone, locale,
			two_factor_enabled, password_changed_at,
			preferences, metadata,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15,
			$16, $17,
			$18, $19,
			$20, $21, $22, $23
		)
	`

	preferencesJSON, err := json.Marshal(u.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	metadataJSON, err := json.Marshal(u.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var createdBy, updatedBy interface{}
	if u.CreatedBy != "" {
		createdBy = u.CreatedBy
	}
	if u.UpdatedBy != "" {
		updatedBy = u.UpdatedBy
	}

	_, err = r.db.ExecContext(ctx, query,
		u.ID, u.TenantID, u.Email, u.Password, u.FirstName, u.LastName, u.Role, u.Status,
		u.EmailVerified, u.EmailVerifiedAt, u.LastLoginAt,
		u.Avatar, u.Phone, u.Timezone, u.Locale,
		u.TwoFactorEnabled, u.PasswordChangedAt,
		preferencesJSON, metadataJSON,
		u.CreatedAt, u.UpdatedAt, createdBy, updatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID (tenant-scoped)
func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, userID string) (*user.User, error) {
	query := `
		SELECT
			id, tenant_id, email, password_hash, first_name, last_name, role, status,
			email_verified, email_verified_at, last_login_at,
			avatar, phone, timezone, locale,
			two_factor_enabled, password_changed_at,
			preferences, metadata,
			created_at, updated_at, created_by, updated_by
		FROM users
		WHERE id = $1 AND tenant_id = $2
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, userID, tenantID))
}

// GetByEmail retrieves a user by email (tenant-scoped)
func (r *PostgresRepository) GetByEmail(ctx context.Context, tenantID, email string) (*user.User, error) {
	query := `
		SELECT
			id, tenant_id, email, password_hash, first_name, last_name, role, status,
			email_verified, email_verified_at, last_login_at,
			avatar, phone, timezone, locale,
			two_factor_enabled, password_changed_at,
			preferences, metadata,
			created_at, updated_at, created_by, updated_by
		FROM users
		WHERE email = $1 AND tenant_id = $2
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, email, tenantID))
}

// GetByEmailGlobal retrieves a user by email across all tenants (for auth)
func (r *PostgresRepository) GetByEmailGlobal(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT
			id, tenant_id, email, password_hash, first_name, last_name, role, status,
			email_verified, email_verified_at, last_login_at,
			avatar, phone, timezone, locale,
			two_factor_enabled, password_changed_at,
			preferences, metadata,
			created_at, updated_at, created_by, updated_by
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, email))
}

// List retrieves all users in a tenant with pagination
func (r *PostgresRepository) List(ctx context.Context, tenantID string, filters ListFilters) ([]*user.User, int64, error) {
	// Build query with filters
	whereClause, args := buildWhereClause(tenantID, filters)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// List query
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, email, password_hash, first_name, last_name, role, status,
			email_verified, email_verified_at, last_login_at,
			avatar, phone, timezone, locale,
			two_factor_enabled, password_changed_at,
			preferences, metadata,
			created_at, updated_at, created_by, updated_by
		FROM users
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, len(args)+1, len(args)+2)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]*user.User, 0)
	for rows.Next() {
		u, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

// Update updates an existing user
func (r *PostgresRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users SET
			email = $3,
			password = $4,
			first_name = $5,
			last_name = $6,
			role = $7,
			status = $8,
			email_verified = $9,
			email_verified_at = $10,
			last_login_at = $11,
			avatar = $12,
			phone = $13,
			timezone = $14,
			locale = $15,
			two_factor_enabled = $16,
			password_changed_at = $17,
			preferences = $18,
			metadata = $19,
			updated_at = $20,
			updated_by = $21
		WHERE id = $1 AND tenant_id = $2
	`

	preferencesJSON, err := json.Marshal(u.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	metadataJSON, err := json.Marshal(u.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	u.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		u.ID, u.TenantID,
		u.Email, u.Password, u.FirstName, u.LastName, u.Role, u.Status,
		u.EmailVerified, u.EmailVerifiedAt, u.LastLoginAt,
		u.Avatar, u.Phone, u.Timezone, u.Locale,
		u.TwoFactorEnabled, u.PasswordChangedAt,
		preferencesJSON, metadataJSON,
		u.UpdatedAt, u.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// Delete soft deletes a user
func (r *PostgresRepository) Delete(ctx context.Context, tenantID, userID string) error {
	query := `
		UPDATE users
		SET status = 'inactive'
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, userID, tenantID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// ExistsByEmail checks if a user with the given email exists in tenant
func (r *PostgresRepository) ExistsByEmail(ctx context.Context, tenantID, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// CountByTenant counts users in a tenant
func (r *PostgresRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE tenant_id = $1`

	var count int64
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// CountByRole counts users by role in a tenant
func (r *PostgresRepository) CountByRole(ctx context.Context, tenantID string, role user.UserRole) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND role = $2`

	var count int64
	err := r.db.QueryRowContext(ctx, query, tenantID, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}

	return count, nil
}

// CountByStatus counts users by status in a tenant
func (r *PostgresRepository) CountByStatus(ctx context.Context, tenantID string, status user.UserStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND status = $2`

	var count int64
	err := r.db.QueryRowContext(ctx, query, tenantID, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by status: %w", err)
	}

	return count, nil
}

// UpdateLastLogin updates user's last login timestamp
func (r *PostgresRepository) UpdateLastLogin(ctx context.Context, tenantID, userID string) error {
	query := `
		UPDATE users
		SET last_login_at = $3, updated_at = $3
		WHERE id = $1 AND tenant_id = $2
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, userID, tenantID, now)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// GetOwner retrieves the tenant owner
func (r *PostgresRepository) GetOwner(ctx context.Context, tenantID string) (*user.User, error) {
	query := `
		SELECT
			id, tenant_id, email, password_hash, first_name, last_name, role, status,
			email_verified, email_verified_at, last_login_at,
			avatar, phone, timezone, locale,
			two_factor_enabled, password_changed_at,
			preferences, metadata,
			created_at, updated_at, created_by, updated_by
		FROM users
		WHERE tenant_id = $1 AND role = 'owner'
		LIMIT 1
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, tenantID))
}

// scanUser scans a user from a row
func (r *PostgresRepository) scanUser(row *sql.Row) (*user.User, error) {
	u := &user.User{}
	var preferencesJSON, metadataJSON []byte

	err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Password, &u.FirstName, &u.LastName, &u.Role, &u.Status,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.LastLoginAt,
		&u.Avatar, &u.Phone, &u.Timezone, &u.Locale,
		&u.TwoFactorEnabled, &u.PasswordChangedAt,
		&preferencesJSON, &metadataJSON,
		&u.CreatedAt, &u.UpdatedAt, &u.CreatedBy, &u.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if err := json.Unmarshal(preferencesJSON, &u.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &u.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return u, nil
}

// scanUserFromRows scans a user from rows
func (r *PostgresRepository) scanUserFromRows(rows *sql.Rows) (*user.User, error) {
	u := &user.User{}
	var preferencesJSON, metadataJSON []byte

	err := rows.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Password, &u.FirstName, &u.LastName, &u.Role, &u.Status,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.LastLoginAt,
		&u.Avatar, &u.Phone, &u.Timezone, &u.Locale,
		&u.TwoFactorEnabled, &u.PasswordChangedAt,
		&preferencesJSON, &metadataJSON,
		&u.CreatedAt, &u.UpdatedAt, &u.CreatedBy, &u.UpdatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if err := json.Unmarshal(preferencesJSON, &u.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &u.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return u, nil
}

// buildWhereClause builds WHERE clause for list queries
func buildWhereClause(tenantID string, filters ListFilters) (string, []interface{}) {
	conditions := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []interface{}{tenantID}
	argCount := 2

	if filters.Role != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argCount))
		args = append(args, *filters.Role)
		argCount++
	}

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)",
			argCount, argCount, argCount,
		))
		args = append(args, searchPattern)
		argCount++
	}

	return strings.Join(conditions, " AND "), args
}
