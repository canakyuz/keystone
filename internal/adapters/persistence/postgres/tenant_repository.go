package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	. "nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/ports/repositories"
)

// TenantRepository implements the tenant repository interface for PostgreSQL
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository creates a new PostgreSQL tenant repository
func NewTenantRepository(db *sql.DB) repositories.TenantRepository {
	return &TenantRepository{
		db: db,
	}
}

// Create creates a new tenant in the database
func (r *TenantRepository) Create(ctx context.Context, tenant *tenant.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, name, slug, custom_domain, status, subscription_id,
			settings, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	var customDomain *string
	if tenant.CustomDomain != nil && tenant.CustomDomain.String() != "" {
		domain := tenant.CustomDomain.String()
		customDomain = &domain
	}
	var subscriptionID *uuid.UUID
	if tenant.SubscriptionID != nil {
		id := uuid.UUID(*tenant.SubscriptionID)
		subscriptionID = &id
	}

	_, err := r.db.ExecContext(ctx, query,
		uuid.UUID(tenant.ID),
		tenant.Name,
		tenant.Slug.String(),
		customDomain,
		string(tenant.Status),
		subscriptionID,
		tenant.Settings,
		tenant.Metadata,
		tenant.CreatedAt.Time(),
		tenant.UpdatedAt.Time(),
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "tenants_slug_key" {
					return ErrTenantSlugAlreadyExists
				}
				if pqErr.Constraint == "tenants_custom_domain_key" {
					return ErrCustomDomainAlreadyExists
				}
			}
		}
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// GetByID retrieves a tenant by ID
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	query := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	return r.scanTenant(row)
}

// GetBySlug retrieves a tenant by slug
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	query := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants
		WHERE slug = $1`

	row := r.db.QueryRowContext(ctx, query, slug)

	return r.scanTenant(row)
}

// GetByCustomDomain retrieves a tenant by custom domain
func (r *TenantRepository) GetByCustomDomain(ctx context.Context, domain string) (*tenant.Tenant, error) {
	query := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants
		WHERE custom_domain = $1`

	row := r.db.QueryRowContext(ctx, query, domain)

	return r.scanTenant(row)
}

// Update updates an existing tenant
func (r *TenantRepository) Update(ctx context.Context, tenant *tenant.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, slug = $3, custom_domain = $4, status = $5,
			subscription_id = $6, settings = $7, metadata = $8, updated_at = $9
		WHERE id = $1`

	var customDomain *string
	if tenant.CustomDomain != nil && tenant.CustomDomain.String() != "" {
		domain := tenant.CustomDomain.String()
		customDomain = &domain
	}

	var subscriptionID *uuid.UUID
	if tenant.SubscriptionID != nil {
		id := uuid.UUID(*tenant.SubscriptionID)
		subscriptionID = &id
	}

	result, err := r.db.ExecContext(ctx, query,
		uuid.UUID(tenant.ID),
		tenant.Name,
		tenant.Slug.String(),
		customDomain,
		string(tenant.Status),
		subscriptionID,
		tenant.Settings,
		tenant.Metadata,
		tenant.UpdatedAt.Time(),
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "tenants_slug_key" {
					return ErrTenantSlugAlreadyExists
				}
				if pqErr.Constraint == "tenants_custom_domain_key" {
					return ErrCustomDomainAlreadyExists
				}
			}
		}
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// Delete soft deletes a tenant (sets status to cancelled)
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE tenants
		SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status != 'cancelled'`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// List retrieves tenants with pagination and filtering
func (r *TenantRepository) List(ctx context.Context, criteria repositories.TenantListCriteria) ([]*tenant.Tenant, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argCount := 0

	if criteria.Status != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, string(*criteria.Status))
	}

	if criteria.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+criteria.Search+"%")
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM tenants " + whereClause
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	// Build main query with pagination
	query := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`

	argCount++
	query = fmt.Sprintf(query, argCount, argCount+1)
	args = append(args, criteria.PageSize, (criteria.Page-1)*criteria.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		tenant, err := r.scanTenant(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return tenants, totalCount, nil
}

// IsSlugAvailable checks if a tenant slug is available
func (r *TenantRepository) IsSlugAvailable(ctx context.Context, slug string) (bool, error) {
	query := "SELECT COUNT(*) FROM tenants WHERE slug = $1"
	args := []interface{}{slug}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check slug availability: %w", err)
	}

	return count == 0, nil
}

// IsDomainAvailable checks if a custom domain is available
func (r *TenantRepository) IsDomainAvailable(ctx context.Context, domain string) (bool, error) {
	query := "SELECT COUNT(*) FROM tenants WHERE custom_domain = $1"
	args := []interface{}{domain}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check domain availability: %w", err)
	}

	return count == 0, nil
}

// scanTenant is a helper function to scan tenant from database row
func (r *TenantRepository) scanTenant(scanner interface {
	Scan(dest ...interface{}) error
}) (*tenant.Tenant, error) {
	var (
		id             uuid.UUID
		name           string
		slug           string
		customDomain   *string
		status         string
		subscriptionID *uuid.UUID
		settings       map[string]interface{}
		metadata       map[string]interface{}
		createdAt      sql.NullTime
		updatedAt      sql.NullTime
	)

	err := scanner.Scan(
		&id, &name, &slug, &customDomain, &status, &subscriptionID,
		&settings, &metadata, &createdAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to scan tenant: %w", err)
	}

	// Create value objects
	tenantSlug, err := NewTenantSlug(slug)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant slug in database: %w", err)
	}

	var domain *Domain
	if customDomain != nil {
		d, err := NewDomain(*customDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid custom domain in database: %w", err)
		}
		domain = d
	}

	tenantStatus, err := tenant.ParseStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant status in database: %w", err)
	}

	var subID *SubscriptionID
	if subscriptionID != nil {
		subID = (*SubscriptionID)(subscriptionID)
	}

	// Create tenant entity
	t := &tenant.Tenant{
		ID:             TenantID(id),
		Name:           name,
		Slug:           *tenantSlug,
		CustomDomain:   domain,
		Status:         tenantStatus,
		SubscriptionID: subID,
		Settings:       settings,
		Metadata:       metadata,
		CreatedAt:      NewTimestampFromTime(createdAt.Time),
		UpdatedAt:      NewTimestampFromTime(updatedAt.Time),
	}

	return t, nil
}

func (r *TenantRepository) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to activate tenant: %w", err)
	}
	return nil
}

func (r *TenantRepository) DeactivateTenant(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET status = 'inactive', updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate tenant: %w", err)
	}
	return nil
}

func (r *TenantRepository) UpdateSettings(ctx context.Context, tenantID uuid.UUID, settings tenant.Settings) error {
	query := `UPDATE tenants SET settings = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, tenantID, settings)
	if err != nil {
		return fmt.Errorf("failed to update tenant settings: %w", err)
	}
	return nil
}

func (r *TenantRepository) Search(ctx context.Context, query string, limit, offset int) ([]*tenant.Tenant, error) {
	sqlQuery := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants
		WHERE name ILIKE $1 OR slug ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	searchPattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, sqlQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		tenant, err := r.scanTenant(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return tenants, nil
}

func (r *TenantRepository) GetTenantsByCreatedDate(ctx context.Context, startDate, endDate string) ([]*tenant.Tenant, error) {
	query := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants by date: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		tenant, err := r.scanTenant(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return tenants, nil
}

func (r *TenantRepository) ListSimple(ctx context.Context, limit, offset int) ([]*tenant.Tenant, error) {
	query := `
		SELECT id, name, slug, custom_domain, status, subscription_id,
			   settings, metadata, created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		tenant, err := r.scanTenant(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return tenants, nil
}

func (r *TenantRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenants").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tenants: %w", err)
	}
	return count, nil
}

func (r *TenantRepository) GetActiveTenantsCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenants WHERE status = 'active'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active tenants: %w", err)
	}
	return count, nil
}
