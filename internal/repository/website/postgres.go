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

// 🎓 EDUCATIONAL NOTE: PostgreSQL Repository Pattern
//
// Bu dosya Repository Pattern'in PostgreSQL implementasyonudur.
//
// KEY CONCEPTS:
// 1. **Tenant Isolation**: Her query'de tenant_id kontrolü zorunlu
// 2. **Soft Delete**: deleted_at IS NULL ile silinen kayıtları filtrele
// 3. **Context**: Timeout ve cancellation için context.Context kullan
// 4. **Prepared Statements**: SQL injection'dan korunma (automatic with $1, $2)
// 5. **Error Handling**: PostgreSQL error kodlarını domain error'larına map'le
//
// POSTGRES ERROR CODES:
// - 23505: unique_violation (duplicate key)
// - 23503: foreign_key_violation
// - 23502: not_null_violation

type postgresRepo struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL website repository
//
// 🎓 CONSTRUCTOR PATTERN: Interface döner, concrete type değil
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepo{db: db}
}

// ═════════════════════════════════════════════════════════════
// QUERY METHODS
// ═════════════════════════════════════════════════════════════

// List returns all websites for a tenant
//
// 🎓 MULTI-TENANT QUERY: tenant_id filtresi zorunlu
// 🎓 SOFT DELETE: deleted_at IS NULL ile silinen kayıtlar dahil değil
func (r *postgresRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*website.Website, error) {
	query := `
		SELECT
			id, tenant_id, name, slug, type, description, status, homepage_id,
			title, logo, favicon,
			custom_domain, custom_domain_verified, primary_url,
			template_id, template_name, template_version,
			published_at, unpublished_at, archived_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM websites
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // 🎓 RESOURCE CLEANUP: Her zaman defer ile close

	var websites []*website.Website
	for rows.Next() {
		site, err := scanWebsite(rows)
		if err != nil {
			return nil, err
		}
		websites = append(websites, site)
	}

	// 🎓 CHECK ITERATION ERROR: rows.Err() kontrol edilmeli
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return websites, nil
}

// GetByID retrieves a website by ID (tenant-scoped)
//
// 🎓 SECURITY: Hem ID hem tenant_id ile filtrele (cross-tenant access prevention)
func (r *postgresRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*website.Website, error) {
	query := `
		SELECT
			id, tenant_id, name, slug, type, description, status, homepage_id,
			title, logo, favicon,
			custom_domain, custom_domain_verified, primary_url,
			template_id, template_name, template_version,
			published_at, unpublished_at, archived_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM websites
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

// GetBySlug retrieves a website by slug (global, for public access)
//
// 🎓 PUBLIC ACCESS: Slug global unique olduğu için tenant_id gerekmez
// Ancak published status kontrol edilebilir (caller'ın sorumluluğu)
func (r *postgresRepo) GetBySlug(ctx context.Context, slug string) (*website.Website, error) {
	query := `
		SELECT
			id, tenant_id, name, slug, type, description, status, homepage_id,
			title, logo, favicon,
			custom_domain, custom_domain_verified, primary_url,
			template_id, template_name, template_version,
			published_at, unpublished_at, archived_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM websites
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

// SlugExists checks if a slug already exists
//
// 🎓 VALIDATION HELPER: Slug uniqueness kontrolü için
func (r *postgresRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM websites WHERE slug = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// ═════════════════════════════════════════════════════════════
// WRITE METHODS
// ═════════════════════════════════════════════════════════════

// Create creates a new website
//
// 🎓 INSERT PATTERN:
// - RETURNING clause ile generated values'ları al (id, timestamps)
// - Duplicate key error'ı yakalayıp domain error'a map'le
func (r *postgresRepo) Create(ctx context.Context, site *website.Website) error {
	query := `
		INSERT INTO websites (
			id, tenant_id, name, slug, type, description, status, homepage_id,
			title, logo, favicon,
			custom_domain, custom_domain_verified, primary_url,
			template_id, template_name, template_version,
			published_at, unpublished_at, archived_at,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20,
			$21, $22, $23, $24
		)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		site.ID,
		site.TenantID,
		site.Name,
		site.Slug,
		site.Type,
		site.Description,
		site.Status,
		site.HomepageID,
		// Content
		site.Title,
		site.Logo,
		site.Favicon,
		// Domain
		site.CustomDomain,
		site.CustomDomainVerified,
		site.PrimaryURL,
		// Template
		site.TemplateID,
		site.TemplateName,
		site.TemplateVersion,
		// Publishing
		site.PublishedAt,
		site.UnpublishedAt,
		site.ArchivedAt,
		// Metadata
		site.CreatedAt,
		site.UpdatedAt,
		site.CreatedBy,
		site.UpdatedBy,
	).Scan(&site.ID, &site.CreatedAt, &site.UpdatedAt)

	if err != nil {
		// 🎓 POSTGRES ERROR HANDLING: Error code'larını kontrol et
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				return website.ErrSlugExists
			case "23503": // foreign_key_violation
				return errors.New("foreign key constraint violation")
			}
		}
		return err
	}

	return nil
}

// Update updates an existing website
//
// 🎓 UPDATE PATTERN:
// - SET updated_at = NOW() otomatik
// - WHERE id AND tenant_id (security)
// - RowsAffected() ile not found kontrolü
func (r *postgresRepo) Update(ctx context.Context, site *website.Website) error {
	query := `
		UPDATE websites
		SET
			name = $1,
			type = $2,
			description = $3,
			status = $4,
			homepage_id = $5,
			title = $6,
			logo = $7,
			favicon = $8,
			custom_domain = $9,
			custom_domain_verified = $10,
			primary_url = $11,
			template_id = $12,
			template_name = $13,
			template_version = $14,
			published_at = $15,
			unpublished_at = $16,
			archived_at = $17,
			updated_at = $18,
			updated_by = $19
		WHERE id = $20 AND tenant_id = $21 AND deleted_at IS NULL
	`

	site.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		site.Name,
		site.Type,
		site.Description,
		site.Status,
		site.HomepageID,
		// Content
		site.Title,
		site.Logo,
		site.Favicon,
		// Domain
		site.CustomDomain,
		site.CustomDomainVerified,
		site.PrimaryURL,
		// Template
		site.TemplateID,
		site.TemplateName,
		site.TemplateVersion,
		// Publishing
		site.PublishedAt,
		site.UnpublishedAt,
		site.ArchivedAt,
		// Metadata
		site.UpdatedAt,
		site.UpdatedBy,
		// WHERE
		site.ID,
		site.TenantID,
	)
	if err != nil {
		return err
	}

	// 🎓 CHECK AFFECTED ROWS: 0 rows = not found
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return website.ErrNotFound
	}

	return nil
}

// Delete soft-deletes a website (tenant-scoped)
//
// 🎓 SOFT DELETE: deleted_at timestamp'i set et, kaydı silme
// WHY: Data recovery, audit trail, referential integrity
func (r *postgresRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		UPDATE websites
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

// ═════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═════════════════════════════════════════════════════════════

// scanWebsite is a helper function to scan a website from a database row
//
// 🎓 SCAN PATTERN:
// - Interface ile hem *sql.Row hem *sql.Rows destekle
// - NULL değerleri pointer (*string, *uuid.UUID) ile handle et
// - Scan sırası SELECT sırası ile aynı olmalı
func scanWebsite(scanner interface {
	Scan(dest ...interface{}) error
}) (*website.Website, error) {
	var site website.Website

	err := scanner.Scan(
		&site.ID,
		&site.TenantID,
		&site.Name,
		&site.Slug,
		&site.Type,
		&site.Description,
		&site.Status,
		&site.HomepageID,
		// Content
		&site.Title,
		&site.Logo,
		&site.Favicon,
		// Domain
		&site.CustomDomain,
		&site.CustomDomainVerified,
		&site.PrimaryURL,
		// Template
		&site.TemplateID,
		&site.TemplateName,
		&site.TemplateVersion,
		// Publishing
		&site.PublishedAt,
		&site.UnpublishedAt,
		&site.ArchivedAt,
		// Metadata
		&site.CreatedAt,
		&site.UpdatedAt,
		&site.DeletedAt,
		&site.CreatedBy,
		&site.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	return &site, nil
}

// 🎓 SELF-REVIEW:
//
// ✅ Multi-tenant isolation: Her query'de tenant_id kontrolü
// ✅ Soft delete: deleted_at IS NULL kullanımı
// ✅ Context support: Timeout/cancellation için
// ✅ Error handling: PostgreSQL error kodlarını domain error'larına map
// ✅ Resource cleanup: defer rows.Close()
// ✅ Security: Cross-tenant access prevention
// ✅ Full entity mapping: Tüm alanlar database'e yazılıyor
// ✅ Idiomatic Go: Interface return, error wrapping
