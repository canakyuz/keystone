package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/template"
	"nexspaces-api/internal/core/ports/repositories"
)

// TemplateRepository implements the template repository interface for PostgreSQL
type TemplateRepository struct {
	db *sql.DB
}

// NewTemplateRepository creates a new PostgreSQL template repository
func NewTemplateRepository(db *sql.DB) repositories.TemplateRepository {
	return &TemplateRepository{
		db: db,
	}
}

// Create creates a new template in the database
func (r *TemplateRepository) Create(ctx context.Context, tmpl *template.Template) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tmpl.TenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		INSERT INTO templates (
			id, tenant_id, created_by, name, description, category, tags,
			configuration, variables, version, status, visibility, pricing_model,
			price_amount, price_currency, install_count, rating, rating_count,
			published_at, archived_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23
		)`

	var priceAmount *int64
	var priceCurrency *string
	if tmpl.PriceAmount != nil {
		amount := tmpl.PriceAmount.Amount()
		currency := tmpl.PriceAmount.Currency()
		priceAmount = &amount
		priceCurrency = &currency
	}

	var publishedAt *sql.NullTime
	if tmpl.PublishedAt != nil {
		publishedAt = &sql.NullTime{
			Time:  tmpl.PublishedAt.Time(),
			Valid: true,
		}
	}

	var archivedAt *sql.NullTime
	if tmpl.ArchivedAt != nil {
		archivedAt = &sql.NullTime{
			Time:  tmpl.ArchivedAt.Time(),
			Valid: true,
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		uuid.UUID(tmpl.ID),
		uuid.UUID(tmpl.TenantID),
		uuid.UUID(tmpl.CreatedBy),
		tmpl.Name,
		tmpl.Description,
		string(tmpl.Category),
		pq.Array(tmpl.Tags),
		tmpl.Configuration,
		tmpl.Variables,
		tmpl.Version.String(),
		string(tmpl.Status),
		string(tmpl.Visibility),
		string(tmpl.PricingModel),
		priceAmount,
		priceCurrency,
		tmpl.InstallCount,
		tmpl.Rating,
		tmpl.RatingCount,
		publishedAt,
		archivedAt,
		tmpl.Metadata,
		tmpl.CreatedAt.Time(),
		tmpl.UpdatedAt.Time(),
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "templates_tenant_id_name_key" {
					return shared.ErrTemplateAlreadyExists
				}
			}
		}
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// GetByID retrieves a template by ID
func (r *TemplateRepository) GetByID(ctx context.Context, id shared.TemplateID) (*template.Template, error) {
	query := `
		SELECT id, tenant_id, created_by, name, description, category, tags,
			   configuration, variables, version, status, visibility, pricing_model,
			   price_amount, price_currency, install_count, rating, rating_count,
			   published_at, archived_at, metadata, created_at, updated_at
		FROM templates
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, uuid.UUID(id))

	return r.scanTemplate(row)
}

// GetByNameAndTenant retrieves a template by name within a tenant
func (r *TemplateRepository) GetByNameAndTenant(ctx context.Context, name string, tenantID shared.TenantID) (*template.Template, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, created_by, name, description, category, tags,
			   configuration, variables, version, status, visibility, pricing_model,
			   price_amount, price_currency, install_count, rating, rating_count,
			   published_at, archived_at, metadata, created_at, updated_at
		FROM templates
		WHERE name = $1 AND tenant_id = $2`

	row := r.db.QueryRowContext(ctx, query, name, uuid.UUID(tenantID))

	return r.scanTemplate(row)
}

// Update updates an existing template
func (r *TemplateRepository) Update(ctx context.Context, tmpl *template.Template) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tmpl.TenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		UPDATE templates
		SET name = $2, description = $3, category = $4, tags = $5,
			configuration = $6, variables = $7, version = $8, status = $9,
			visibility = $10, pricing_model = $11, price_amount = $12,
			price_currency = $13, install_count = $14, rating = $15,
			rating_count = $16, published_at = $17, archived_at = $18,
			metadata = $19, updated_at = $20
		WHERE id = $1`

	var priceAmount *int64
	var priceCurrency *string
	if tmpl.PriceAmount != nil {
		amount := tmpl.PriceAmount.Amount()
		currency := tmpl.PriceAmount.Currency()
		priceAmount = &amount
		priceCurrency = &currency
	}

	var publishedAt *sql.NullTime
	if tmpl.PublishedAt != nil {
		publishedAt = &sql.NullTime{
			Time:  tmpl.PublishedAt.Time(),
			Valid: true,
		}
	}

	var archivedAt *sql.NullTime
	if tmpl.ArchivedAt != nil {
		archivedAt = &sql.NullTime{
			Time:  tmpl.ArchivedAt.Time(),
			Valid: true,
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		uuid.UUID(tmpl.ID),
		tmpl.Name,
		tmpl.Description,
		string(tmpl.Category),
		pq.Array(tmpl.Tags),
		tmpl.Configuration,
		tmpl.Variables,
		tmpl.Version.String(),
		string(tmpl.Status),
		string(tmpl.Visibility),
		string(tmpl.PricingModel),
		priceAmount,
		priceCurrency,
		tmpl.InstallCount,
		tmpl.Rating,
		tmpl.RatingCount,
		publishedAt,
		archivedAt,
		tmpl.Metadata,
		tmpl.UpdatedAt.Time(),
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "templates_tenant_id_name_key" {
					return shared.ErrTemplateAlreadyExists
				}
			}
		}
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrTemplateNotFound
	}

	return nil
}

// Delete soft deletes a template (sets status to archived)
func (r *TemplateRepository) Delete(ctx context.Context, id shared.TemplateID, tenantID shared.TenantID) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		UPDATE templates
		SET status = 'archived', archived_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status != 'archived'`

	result, err := r.db.ExecContext(ctx, query, uuid.UUID(id))
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrTemplateNotFound
	}

	return nil
}

// Search searches templates with filtering and pagination
func (r *TemplateRepository) Search(ctx context.Context, criteria repositories.TemplateSearchCriteria) ([]*template.Template, int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, criteria.TenantID); err != nil {
		return nil, 0, fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Build WHERE clause for search
	whereClause := "WHERE (tenant_id = $1 OR (visibility = 'public' AND status = 'published'))"
	args := []interface{}{uuid.UUID(criteria.TenantID)}
	argCount := 1

	// Add search query
	if criteria.Query != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (to_tsvector('english', name || ' ' || description) @@ plainto_tsquery('english', $%d))", argCount)
		args = append(args, criteria.Query)
	}

	// Add category filter
	if criteria.Category != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, string(*criteria.Category))
	}

	// Add visibility filter
	if criteria.Visibility != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND visibility = $%d", argCount)
		args = append(args, string(*criteria.Visibility))
	}

	// Add pricing model filter
	if criteria.PricingModel != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND pricing_model = $%d", argCount)
		args = append(args, string(*criteria.PricingModel))
	}

	// Add tags filter
	if len(criteria.Tags) > 0 {
		argCount++
		whereClause += fmt.Sprintf(" AND tags && $%d", argCount)
		args = append(args, pq.Array(criteria.Tags))
	}

	// Add status filter (only published templates visible in search)
	whereClause += " AND status = 'published'"

	// Count total records
	countQuery := "SELECT COUNT(*) FROM templates " + whereClause
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Build main query with pagination and ordering
	orderBy := "ORDER BY rating DESC, install_count DESC, published_at DESC"
	query := fmt.Sprintf(`
		SELECT id, tenant_id, created_by, name, description, category, tags,
			   configuration, variables, version, status, visibility, pricing_model,
			   price_amount, price_currency, install_count, rating, rating_count,
			   published_at, archived_at, metadata, created_at, updated_at
		FROM templates %s %s
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, argCount+1, argCount+2)

	args = append(args, criteria.PageSize, (criteria.Page-1)*criteria.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search templates: %w", err)
	}
	defer rows.Close()

	var templates []*template.Template
	for rows.Next() {
		tmpl, err := r.scanTemplate(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, tmpl)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return templates, totalCount, nil
}

// ListByTenant retrieves templates within a tenant
func (r *TemplateRepository) ListByTenant(ctx context.Context, tenantID shared.TenantID, criteria repositories.TemplateListCriteria) ([]*template.Template, int, error) {
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

	if criteria.Category != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, string(*criteria.Category))
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM templates " + whereClause
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT id, tenant_id, created_by, name, description, category, tags,
			   configuration, variables, version, status, visibility, pricing_model,
			   price_amount, price_currency, install_count, rating, rating_count,
			   published_at, archived_at, metadata, created_at, updated_at
		FROM templates %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argCount+1, argCount+2)

	args = append(args, criteria.PageSize, (criteria.Page-1)*criteria.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []*template.Template
	for rows.Next() {
		tmpl, err := r.scanTemplate(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, tmpl)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return templates, totalCount, nil
}

// CreateInstallation creates a new template installation
func (r *TemplateRepository) CreateInstallation(ctx context.Context, installation *template.Installation) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, installation.TenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		INSERT INTO template_installations (
			id, template_id, tenant_id, installed_by, configuration,
			status, installed_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err := r.db.ExecContext(ctx, query,
		uuid.UUID(installation.ID),
		uuid.UUID(installation.TemplateID),
		uuid.UUID(installation.TenantID),
		uuid.UUID(installation.InstalledBy),
		installation.Configuration,
		string(installation.Status),
		installation.CreatedAt.Time(),
		installation.Metadata,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "template_installations_template_id_tenant_id_key" {
					return shared.ErrTemplateAlreadyInstalled
				}
			}
		}
		return fmt.Errorf("failed to create installation: %w", err)
	}

	return nil
}

// GetInstallationByTemplateAndTenant retrieves an installation by template and tenant
func (r *TemplateRepository) GetInstallationByTemplateAndTenant(ctx context.Context, templateID shared.TemplateID, tenantID shared.TenantID) (*template.Installation, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, template_id, tenant_id, installed_by, configuration,
			   status, installed_at, uninstalled_at, metadata
		FROM template_installations
		WHERE template_id = $1 AND tenant_id = $2`

	row := r.db.QueryRowContext(ctx, query, uuid.UUID(templateID), uuid.UUID(tenantID))

	return r.scanInstallation(row)
}

// setTenantContext sets the tenant context for Row Level Security
func (r *TemplateRepository) setTenantContext(ctx context.Context, tenantID shared.TenantID) error {
	query := "SELECT set_tenant_context($1)"
	_, err := r.db.ExecContext(ctx, query, uuid.UUID(tenantID))
	return err
}

// scanTemplate is a helper function to scan template from database row
func (r *TemplateRepository) scanTemplate(scanner interface {
	Scan(dest ...interface{}) error
}) (*template.Template, error) {
	var (
		id            uuid.UUID
		tenantID      uuid.UUID
		createdBy     uuid.UUID
		name          string
		description   string
		category      string
		tags          pq.StringArray
		configuration map[string]interface{}
		variables     map[string]interface{}
		version       string
		status        string
		visibility    string
		pricingModel  string
		priceAmount   *int64
		priceCurrency *string
		installCount  int64
		rating        float64
		ratingCount   int64
		publishedAt   sql.NullTime
		archivedAt    sql.NullTime
		metadata      map[string]interface{}
		createdAt     sql.NullTime
		updatedAt     sql.NullTime
	)

	err := scanner.Scan(
		&id, &tenantID, &createdBy, &name, &description, &category, &tags,
		&configuration, &variables, &version, &status, &visibility, &pricingModel,
		&priceAmount, &priceCurrency, &installCount, &rating, &ratingCount,
		&publishedAt, &archivedAt, &metadata, &createdAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to scan template: %w", err)
	}

	// Parse domain enums
	templateCategory, err := template.ParseCategory(category)
	if err != nil {
		return nil, fmt.Errorf("invalid template category in database: %w", err)
	}

	templateStatus, err := template.ParseStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid template status in database: %w", err)
	}

	templateVisibility, err := template.ParseVisibility(visibility)
	if err != nil {
		return nil, fmt.Errorf("invalid template visibility in database: %w", err)
	}

	templatePricingModel, err := template.ParsePricingModel(pricingModel)
	if err != nil {
		return nil, fmt.Errorf("invalid template pricing model in database: %w", err)
	}

	templateVersion, err := shared.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid template version in database: %w", err)
	}

	// Create price amount if present
	var price *shared.Money
	if priceAmount != nil && priceCurrency != nil {
		price, err = shared.NewMoney(*priceAmount, *priceCurrency)
		if err != nil {
			return nil, fmt.Errorf("invalid price in database: %w", err)
		}
	}

	// Handle optional timestamps
	var published *shared.Timestamp
	if publishedAt.Valid {
		published = shared.NewTimestampFromTime(publishedAt.Time)
	}

	var archived *shared.Timestamp
	if archivedAt.Valid {
		archived = shared.NewTimestampFromTime(archivedAt.Time)
	}

	// Create template entity
	tmpl := &template.Template{
		ID:            shared.TemplateID(id),
		TenantID:      shared.TenantID(tenantID),
		CreatedBy:     shared.UserID(createdBy),
		Name:          name,
		Description:   description,
		Category:      templateCategory,
		Tags:          []string(tags),
		Configuration: configuration,
		Variables:     variables,
		Version:       *templateVersion,
		Status:        templateStatus,
		Visibility:    templateVisibility,
		PricingModel:  templatePricingModel,
		PriceAmount:   price,
		InstallCount:  installCount,
		Rating:        rating,
		RatingCount:   ratingCount,
		PublishedAt:   published,
		ArchivedAt:    archived,
		Metadata:      metadata,
		CreatedAt:     shared.NewTimestampFromTime(createdAt.Time),
		UpdatedAt:     shared.NewTimestampFromTime(updatedAt.Time),
	}

	return tmpl, nil
}

// scanInstallation is a helper function to scan installation from database row
func (r *TemplateRepository) scanInstallation(scanner interface {
	Scan(dest ...interface{}) error
}) (*template.Installation, error) {
	var (
		id            uuid.UUID
		templateID    uuid.UUID
		tenantID      uuid.UUID
		installedBy   uuid.UUID
		configuration map[string]interface{}
		status        string
		installedAt   sql.NullTime
		uninstalledAt sql.NullTime
		metadata      map[string]interface{}
	)

	err := scanner.Scan(
		&id, &templateID, &tenantID, &installedBy, &configuration,
		&status, &installedAt, &uninstalledAt, &metadata,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrInstallationNotFound
		}
		return nil, fmt.Errorf("failed to scan installation: %w", err)
	}

	installationStatus, err := template.ParseInstallationStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid installation status in database: %w", err)
	}

	var uninstalled *shared.Timestamp
	if uninstalledAt.Valid {
		uninstalled = shared.NewTimestampFromTime(uninstalledAt.Time)
	}

	// Create installation entity
	installation := &template.Installation{
		ID:            shared.InstallationID(id),
		TemplateID:    shared.TemplateID(templateID),
		TenantID:      shared.TenantID(tenantID),
		InstalledBy:   shared.UserID(installedBy),
		Configuration: configuration,
		Status:        installationStatus,
		CreatedAt:     shared.NewTimestampFromTime(installedAt.Time),
		UninstalledAt: uninstalled,
		Metadata:      metadata,
	}

	return installation, nil
}
