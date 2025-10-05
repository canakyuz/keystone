package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nexpaces-api/internal/domain/tenant"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new tenant
func (r *PostgresRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, name, slug, email, phone, schema_name,
			status, plan,
			subscription_start, subscription_end, trial_ends_at,
			custom_domain, custom_domain_verified, custom_domain_verified_at,
			settings, metadata,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16,
			$17, $18, $19, $20
		)
	`

	settingsJSON, err := json.Marshal(t.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		t.ID, t.Name, t.Slug, t.Email, t.Phone, t.SchemaName,
		t.Status, t.Plan,
		t.SubscriptionStart, t.SubscriptionEnd, t.TrialEndsAt,
		t.CustomDomain, t.CustomDomainVerified, t.CustomDomainVerifiedAt,
		settingsJSON, metadataJSON,
		t.CreatedAt, t.UpdatedAt, t.CreatedBy, t.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// GetByID retrieves a tenant by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	query := `
		SELECT
			id, name, slug, email, phone, schema_name,
			status, plan,
			subscription_start, subscription_end, trial_ends_at,
			custom_domain, custom_domain_verified, custom_domain_verified_at,
			settings, metadata,
			created_at, updated_at, created_by, updated_by
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`

	t := &tenant.Tenant{}
	var settingsJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.SchemaName,
		&t.Status, &t.Plan,
		&t.SubscriptionStart, &t.SubscriptionEnd, &t.TrialEndsAt,
		&t.CustomDomain, &t.CustomDomainVerified, &t.CustomDomainVerifiedAt,
		&settingsJSON, &metadataJSON,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, tenant.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return t, nil
}

// GetBySlug retrieves a tenant by slug
func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	query := `
		SELECT
			id, name, slug, email, phone, schema_name,
			status, plan,
			subscription_start, subscription_end, trial_ends_at,
			custom_domain, custom_domain_verified, custom_domain_verified_at,
			settings, metadata,
			created_at, updated_at, created_by, updated_by
		FROM tenants
		WHERE slug = $1 AND deleted_at IS NULL
	`

	t := &tenant.Tenant{}
	var settingsJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.SchemaName,
		&t.Status, &t.Plan,
		&t.SubscriptionStart, &t.SubscriptionEnd, &t.TrialEndsAt,
		&t.CustomDomain, &t.CustomDomainVerified, &t.CustomDomainVerifiedAt,
		&settingsJSON, &metadataJSON,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, tenant.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return t, nil
}

// GetByEmail retrieves a tenant by email
func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*tenant.Tenant, error) {
	query := `
		SELECT
			id, name, slug, email, phone, schema_name,
			status, plan,
			subscription_start, subscription_end, trial_ends_at,
			custom_domain, custom_domain_verified, custom_domain_verified_at,
			settings, metadata,
			created_at, updated_at, created_by, updated_by
		FROM tenants
		WHERE email = $1 AND deleted_at IS NULL
	`

	t := &tenant.Tenant{}
	var settingsJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.SchemaName,
		&t.Status, &t.Plan,
		&t.SubscriptionStart, &t.SubscriptionEnd, &t.TrialEndsAt,
		&t.CustomDomain, &t.CustomDomainVerified, &t.CustomDomainVerifiedAt,
		&settingsJSON, &metadataJSON,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, tenant.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by email: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return t, nil
}

// GetByCustomDomain retrieves a tenant by custom domain
func (r *PostgresRepository) GetByCustomDomain(ctx context.Context, domain string) (*tenant.Tenant, error) {
	query := `
		SELECT
			id, name, slug, email, phone, schema_name,
			status, plan,
			subscription_start, subscription_end, trial_ends_at,
			custom_domain, custom_domain_verified, custom_domain_verified_at,
			settings, metadata,
			created_at, updated_at, created_by, updated_by
		FROM tenants
		WHERE custom_domain = $1 AND custom_domain_verified = TRUE AND deleted_at IS NULL
	`

	t := &tenant.Tenant{}
	var settingsJSON, metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, domain).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.SchemaName,
		&t.Status, &t.Plan,
		&t.SubscriptionStart, &t.SubscriptionEnd, &t.TrialEndsAt,
		&t.CustomDomain, &t.CustomDomainVerified, &t.CustomDomainVerifiedAt,
		&settingsJSON, &metadataJSON,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, tenant.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by custom domain: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return t, nil
}

// List retrieves all tenants with pagination and filters
func (r *PostgresRepository) List(ctx context.Context, filters ListFilters) ([]*tenant.Tenant, int64, error) {
	// Build query with filters
	whereClause, args := buildWhereClause(filters)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenants WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
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
			id, name, slug, email, phone, schema_name,
			status, plan,
			subscription_start, subscription_end, trial_ends_at,
			custom_domain, custom_domain_verified, custom_domain_verified_at,
			settings, metadata,
			created_at, updated_at, created_by, updated_by
		FROM tenants
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, len(args)+1, len(args)+2)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	tenants := make([]*tenant.Tenant, 0)
	for rows.Next() {
		t := &tenant.Tenant{}
		var settingsJSON, metadataJSON []byte

		err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.SchemaName,
			&t.Status, &t.Plan,
			&t.SubscriptionStart, &t.SubscriptionEnd, &t.TrialEndsAt,
			&t.CustomDomain, &t.CustomDomainVerified, &t.CustomDomainVerifiedAt,
			&settingsJSON, &metadataJSON,
			&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant: %w", err)
		}

		if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal settings: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		tenants = append(tenants, t)
	}

	return tenants, total, nil
}

// Update updates an existing tenant
func (r *PostgresRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	query := `
		UPDATE tenants SET
			name = $2,
			slug = $3,
			email = $4,
			phone = $5,
			status = $6,
			plan = $7,
			subscription_start = $8,
			subscription_end = $9,
			trial_ends_at = $10,
			custom_domain = $11,
			custom_domain_verified = $12,
			custom_domain_verified_at = $13,
			settings = $14,
			metadata = $15,
			updated_at = $16,
			updated_by = $17
		WHERE id = $1 AND deleted_at IS NULL
	`

	settingsJSON, err := json.Marshal(t.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	t.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		t.ID,
		t.Name, t.Slug, t.Email, t.Phone,
		t.Status, t.Plan,
		t.SubscriptionStart, t.SubscriptionEnd, t.TrialEndsAt,
		t.CustomDomain, t.CustomDomainVerified, t.CustomDomainVerifiedAt,
		settingsJSON, metadataJSON,
		t.UpdatedAt, t.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return tenant.ErrTenantNotFound
	}

	return nil
}

// Delete soft deletes a tenant
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE tenants
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return tenant.ErrTenantNotFound
	}

	return nil
}

// ExistsBySlug checks if a tenant with the given slug exists
func (r *PostgresRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant existence by slug: %w", err)
	}

	return exists, nil
}

// ExistsByEmail checks if a tenant with the given email exists
func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE email = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant existence by email: %w", err)
	}

	return exists, nil
}

// ExistsByCustomDomain checks if a tenant with the given custom domain exists
func (r *PostgresRepository) ExistsByCustomDomain(ctx context.Context, domain string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE custom_domain = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, domain).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant existence by custom domain: %w", err)
	}

	return exists, nil
}

// CountByStatus counts tenants by status
func (r *PostgresRepository) CountByStatus(ctx context.Context, status tenant.TenantStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM tenants WHERE status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tenants by status: %w", err)
	}

	return count, nil
}

// CountByPlan counts tenants by subscription plan
func (r *PostgresRepository) CountByPlan(ctx context.Context, plan tenant.SubscriptionPlan) (int64, error) {
	query := `SELECT COUNT(*) FROM tenants WHERE plan = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, plan).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tenants by plan: %w", err)
	}

	return count, nil
}

// SetCustomDomain sets a custom domain for a tenant
func (r *PostgresRepository) SetCustomDomain(ctx context.Context, tenantID, domain string) error {
	query := `
		UPDATE tenants
		SET custom_domain = $2,
		    custom_domain_verified = FALSE,
		    custom_domain_verified_at = NULL,
		    updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, tenantID, domain, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set custom domain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return tenant.ErrTenantNotFound
	}

	return nil
}

// VerifyCustomDomain marks a tenant's custom domain as verified
func (r *PostgresRepository) VerifyCustomDomain(ctx context.Context, tenantID string) error {
	query := `
		UPDATE tenants
		SET custom_domain_verified = TRUE,
		    custom_domain_verified_at = $2,
		    updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, tenantID, now)
	if err != nil {
		return fmt.Errorf("failed to verify custom domain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return tenant.ErrTenantNotFound
	}

	return nil
}

// GetStats returns tenant statistics
func (r *PostgresRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'suspended') as suspended,
			COUNT(*) FILTER (WHERE status = 'trial') as trial,
			COUNT(*) FILTER (WHERE plan = 'free') as free_plan,
			COUNT(*) FILTER (WHERE plan = 'starter') as starter_plan,
			COUNT(*) FILTER (WHERE plan = 'pro') as pro_plan,
			COUNT(*) FILTER (WHERE plan = 'enterprise') as enterprise_plan
		FROM tenants
		WHERE deleted_at IS NULL
	`

	var total, active, suspended, trial int64
	var freePlan, starterPlan, proPlan, enterprisePlan int64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&total, &active, &suspended, &trial,
		&freePlan, &starterPlan, &proPlan, &enterprisePlan,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := map[string]interface{}{
		"total":      total,
		"active":     active,
		"suspended":  suspended,
		"trial":      trial,
		"free":       freePlan,
		"starter":    starterPlan,
		"pro":        proPlan,
		"enterprise": enterprisePlan,
	}

	return stats, nil
}

// buildWhereClause builds WHERE clause for list queries
func buildWhereClause(filters ListFilters) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argCount := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}

	if filters.Plan != nil {
		conditions = append(conditions, fmt.Sprintf("plan = $%d", argCount))
		args = append(args, *filters.Plan)
		argCount++
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR slug ILIKE $%d OR email ILIKE $%d)",
			argCount, argCount, argCount,
		))
		args = append(args, searchPattern)
		argCount++
	}

	return strings.Join(conditions, " AND "), args
}
