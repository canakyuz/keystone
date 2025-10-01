package website

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"nexpaces-api/internal/domain/website"
)

type postgresRepo struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL website repository
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*website.Website, error) {
	query := `
		SELECT id, tenant_id, name, slug, description, status, homepage_id, created_at, updated_at, deleted_at
		FROM sites
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var websites []*website.Website
	for rows.Next() {
		site, err := scanWebsite(rows)
		if err != nil {
			return nil, err
		}
		websites = append(websites, site)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return websites, nil
}

func (r *postgresRepo) Create(ctx context.Context, site *website.Website) error {
	query := `
		INSERT INTO sites (id, tenant_id, name, slug, description, status, homepage_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	// Set defaults
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	if site.CreatedAt.IsZero() {
		site.CreatedAt = time.Now()
	}
	if site.UpdatedAt.IsZero() {
		site.UpdatedAt = time.Now()
	}
	if site.Status == "" {
		site.Status = website.StatusDraft
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		site.ID,
		site.TenantID,
		site.Name,
		site.Slug,
		site.Description,
		site.Status,
		site.HomepageID,
		site.CreatedAt,
		site.UpdatedAt,
	).Scan(&site.ID, &site.CreatedAt, &site.UpdatedAt)

	if err != nil {
		// Handle duplicate slug error
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return website.ErrSlugExists
		}
		return err
	}

	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*website.Website, error) {
	query := `
		SELECT id, tenant_id, name, slug, description, status, homepage_id, created_at, updated_at, deleted_at
		FROM sites
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	row := r.db.QueryRowContext(ctx, query, id, tenantID)
	site, err := scanWebsite(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, website.ErrNotFound
		}
		return nil, err
	}

	return site, nil
}

func (r *postgresRepo) GetBySlug(ctx context.Context, slug string) (*website.Website, error) {
	query := `
		SELECT id, tenant_id, name, slug, description, status, homepage_id, created_at, updated_at, deleted_at
		FROM sites
		WHERE slug = $1 AND deleted_at IS NULL
	`

	row := r.db.QueryRowContext(ctx, query, slug)
	site, err := scanWebsite(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, website.ErrNotFound
		}
		return nil, err
	}

	return site, nil
}

func (r *postgresRepo) Update(ctx context.Context, site *website.Website) error {
	query := `
		UPDATE sites
		SET name = $1, description = $2, status = $3, homepage_id = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7 AND deleted_at IS NULL
	`

	site.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		site.Name,
		site.Description,
		site.Status,
		site.HomepageID,
		site.UpdatedAt,
		site.ID,
		site.TenantID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return website.ErrNotFound
	}

	return nil
}

func (r *postgresRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		UPDATE sites
		SET deleted_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, tenantID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return website.ErrNotFound
	}

	return nil
}

func (r *postgresRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM sites WHERE slug = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// scanWebsite is a helper function to scan a website from a database row
func scanWebsite(scanner interface {
	Scan(dest ...interface{}) error
}) (*website.Website, error) {
	var site website.Website
	err := scanner.Scan(
		&site.ID,
		&site.TenantID,
		&site.Name,
		&site.Slug,
		&site.Description,
		&site.Status,
		&site.HomepageID,
		&site.CreatedAt,
		&site.UpdatedAt,
		&site.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &site, nil
}
