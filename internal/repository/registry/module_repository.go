package registry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"nexpaces-api/internal/domain/registry"
)

type moduleRepository struct {
	db *sql.DB
}

// NewModuleRepository creates a new module repository
func NewModuleRepository(db *sql.DB) registry.ModuleRepository {
	return &moduleRepository{db: db}
}

// Create creates a new module
func (r *moduleRepository) Create(ctx context.Context, module *registry.Module) error {
	query := `
		INSERT INTO modules (
			id, name, slug, code, display_name, description,
			category, module_type, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			requires_database, requires_storage, requires_email, database_tables,
			default_limits, installation_notes, configuration_schema, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33,
			$34, $35, $36, $37, $38, $39, $40
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		module.ID, module.Name, module.Slug, module.Code, module.DisplayName, module.Description,
		module.Category, module.ModuleType, module.Status, module.IsPublic, module.IsBeta,
		module.Version, module.MinPlatformVersion, module.PricingModel, module.BasePrice, module.Currency, module.BillingCycle,
		module.Features, module.Capabilities, module.Icon, module.CoverImage, module.Screenshots, module.DemoURL, module.DocumentationURL,
		module.RequiresDatabase, module.RequiresStorage, module.RequiresEmail, module.DatabaseTables,
		module.DefaultLimits, module.InstallationNotes, module.ConfigurationSchema, module.Tags, module.Metadata,
		module.InstallCount, module.Rating, module.ReviewCount,
		module.CreatedAt, module.UpdatedAt, module.CreatedBy, module.UpdatedBy,
	)

	return err
}

// GetByID retrieves a module by ID
func (r *moduleRepository) GetByID(ctx context.Context, id string) (*registry.Module, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, module_type, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			requires_database, requires_storage, requires_email, database_tables,
			default_limits, installation_notes, configuration_schema, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM modules
		WHERE id = $1 AND deleted_at IS NULL
	`

	module := &registry.Module{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&module.ID, &module.Name, &module.Slug, &module.Code, &module.DisplayName, &module.Description,
		&module.Category, &module.ModuleType, &module.Status, &module.IsPublic, &module.IsBeta,
		&module.Version, &module.MinPlatformVersion, &module.PricingModel, &module.BasePrice, &module.Currency, &module.BillingCycle,
		&module.Features, &module.Capabilities, &module.Icon, &module.CoverImage, &module.Screenshots, &module.DemoURL, &module.DocumentationURL,
		&module.RequiresDatabase, &module.RequiresStorage, &module.RequiresEmail, &module.DatabaseTables,
		&module.DefaultLimits, &module.InstallationNotes, &module.ConfigurationSchema, &module.Tags, &module.Metadata,
		&module.InstallCount, &module.Rating, &module.ReviewCount,
		&module.CreatedAt, &module.UpdatedAt, &module.DeletedAt, &module.CreatedBy, &module.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrModuleNotFound
	}

	return module, err
}

// GetByCode retrieves a module by code
func (r *moduleRepository) GetByCode(ctx context.Context, code string) (*registry.Module, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, module_type, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			requires_database, requires_storage, requires_email, database_tables,
			default_limits, installation_notes, configuration_schema, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM modules
		WHERE code = $1 AND deleted_at IS NULL
	`

	module := &registry.Module{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&module.ID, &module.Name, &module.Slug, &module.Code, &module.DisplayName, &module.Description,
		&module.Category, &module.ModuleType, &module.Status, &module.IsPublic, &module.IsBeta,
		&module.Version, &module.MinPlatformVersion, &module.PricingModel, &module.BasePrice, &module.Currency, &module.BillingCycle,
		&module.Features, &module.Capabilities, &module.Icon, &module.CoverImage, &module.Screenshots, &module.DemoURL, &module.DocumentationURL,
		&module.RequiresDatabase, &module.RequiresStorage, &module.RequiresEmail, &module.DatabaseTables,
		&module.DefaultLimits, &module.InstallationNotes, &module.ConfigurationSchema, &module.Tags, &module.Metadata,
		&module.InstallCount, &module.Rating, &module.ReviewCount,
		&module.CreatedAt, &module.UpdatedAt, &module.DeletedAt, &module.CreatedBy, &module.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrModuleNotFound
	}

	return module, err
}

// GetBySlug retrieves a module by slug
func (r *moduleRepository) GetBySlug(ctx context.Context, slug string) (*registry.Module, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, module_type, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			requires_database, requires_storage, requires_email, database_tables,
			default_limits, installation_notes, configuration_schema, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM modules
		WHERE slug = $1 AND deleted_at IS NULL
	`

	module := &registry.Module{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&module.ID, &module.Name, &module.Slug, &module.Code, &module.DisplayName, &module.Description,
		&module.Category, &module.ModuleType, &module.Status, &module.IsPublic, &module.IsBeta,
		&module.Version, &module.MinPlatformVersion, &module.PricingModel, &module.BasePrice, &module.Currency, &module.BillingCycle,
		&module.Features, &module.Capabilities, &module.Icon, &module.CoverImage, &module.Screenshots, &module.DemoURL, &module.DocumentationURL,
		&module.RequiresDatabase, &module.RequiresStorage, &module.RequiresEmail, &module.DatabaseTables,
		&module.DefaultLimits, &module.InstallationNotes, &module.ConfigurationSchema, &module.Tags, &module.Metadata,
		&module.InstallCount, &module.Rating, &module.ReviewCount,
		&module.CreatedAt, &module.UpdatedAt, &module.DeletedAt, &module.CreatedBy, &module.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrModuleNotFound
	}

	return module, err
}

// Update updates a module
func (r *moduleRepository) Update(ctx context.Context, module *registry.Module) error {
	query := `
		UPDATE modules SET
			name = $2, slug = $3, display_name = $4, description = $5,
			category = $6, module_type = $7, status = $8, is_public = $9, is_beta = $10,
			version = $11, pricing_model = $12, base_price = $13, billing_cycle = $14,
			features = $15, capabilities = $16, icon = $17, cover_image = $18,
			default_limits = $19, tags = $20, metadata = $21,
			rating = $22, updated_at = $23, updated_by = $24
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query,
		module.ID, module.Name, module.Slug, module.DisplayName, module.Description,
		module.Category, module.ModuleType, module.Status, module.IsPublic, module.IsBeta,
		module.Version, module.PricingModel, module.BasePrice, module.BillingCycle,
		module.Features, module.Capabilities, module.Icon, module.CoverImage,
		module.DefaultLimits, module.Tags, module.Metadata,
		module.Rating, module.UpdatedAt, module.UpdatedBy,
	)

	return err
}

// Delete soft deletes a module
func (r *moduleRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE modules SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List retrieves modules with filters
func (r *moduleRepository) List(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, module_type, status, is_public, is_beta,
			version, pricing_model, base_price, currency,
			icon, install_count, rating, review_count,
			created_at, updated_at
		FROM modules
		WHERE deleted_at IS NULL
	`

	conditions, args := r.buildFilterConditions(filters)
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += r.buildOrderBy(filters)
	query += r.buildPagination(filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []*registry.Module
	for rows.Next() {
		module := &registry.Module{}
		err := rows.Scan(
			&module.ID, &module.Name, &module.Slug, &module.Code, &module.DisplayName, &module.Description,
			&module.Category, &module.ModuleType, &module.Status, &module.IsPublic, &module.IsBeta,
			&module.Version, &module.PricingModel, &module.BasePrice, &module.Currency,
			&module.Icon, &module.InstallCount, &module.Rating, &module.ReviewCount,
			&module.CreatedAt, &module.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, rows.Err()
}

// ListPublic retrieves public modules
func (r *moduleRepository) ListPublic(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	filters.IsPublic = boolPtr(true)
	filters.Status = statusPtr(registry.ModuleStatusActive)
	return r.List(ctx, filters)
}

// Search searches modules
func (r *moduleRepository) Search(ctx context.Context, query string, filters registry.ModuleFilters) ([]*registry.Module, error) {
	// Simple implementation - can be enhanced with full-text search
	searchQuery := `
		SELECT id, name, slug, code, display_name, description,
			category, module_type, status, is_public, is_beta,
			version, pricing_model, base_price, currency,
			icon, install_count, rating, review_count,
			created_at, updated_at
		FROM modules
		WHERE deleted_at IS NULL
		  AND (name ILIKE $1 OR description ILIKE $1 OR ARRAY_TO_STRING(tags, ' ') ILIKE $1)
	`

	// $1 is used for search pattern, so filter conditions start at $2
	conditions, args := r.buildFilterConditionsWithOffset(filters, 2)
	searchPattern := "%" + query + "%"
	allArgs := append([]interface{}{searchPattern}, args...)

	if len(conditions) > 0 {
		searchQuery += " AND " + strings.Join(conditions, " AND ")
	}

	searchQuery += r.buildOrderBy(filters)
	searchQuery += r.buildPagination(filters)

	rows, err := r.db.QueryContext(ctx, searchQuery, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []*registry.Module
	for rows.Next() {
		module := &registry.Module{}
		err := rows.Scan(
			&module.ID, &module.Name, &module.Slug, &module.Code, &module.DisplayName, &module.Description,
			&module.Category, &module.ModuleType, &module.Status, &module.IsPublic, &module.IsBeta,
			&module.Version, &module.PricingModel, &module.BasePrice, &module.Currency,
			&module.Icon, &module.InstallCount, &module.Rating, &module.ReviewCount,
			&module.CreatedAt, &module.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, rows.Err()
}

// GetInstallCount retrieves install count
func (r *moduleRepository) GetInstallCount(ctx context.Context, moduleID string) (int, error) {
	var count int
	query := `SELECT install_count FROM modules WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, moduleID).Scan(&count)
	return count, err
}

// UpdateInstallCount updates install count
func (r *moduleRepository) UpdateInstallCount(ctx context.Context, moduleID string, delta int) error {
	query := `UPDATE modules SET install_count = install_count + $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, moduleID, delta)
	return err
}

// Helper functions
func (r *moduleRepository) buildFilterConditions(filters registry.ModuleFilters) ([]string, []interface{}) {
	return r.buildFilterConditionsWithOffset(filters, 1)
}

func (r *moduleRepository) buildFilterConditionsWithOffset(filters registry.ModuleFilters, startParam int) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	paramCount := startParam

	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", paramCount))
		args = append(args, *filters.Category)
		paramCount++
	}

	if filters.ModuleType != nil {
		conditions = append(conditions, fmt.Sprintf("module_type = $%d", paramCount))
		args = append(args, *filters.ModuleType)
		paramCount++
	}

	if filters.PricingModel != nil {
		conditions = append(conditions, fmt.Sprintf("pricing_model = $%d", paramCount))
		args = append(args, *filters.PricingModel)
		paramCount++
	}

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", paramCount))
		args = append(args, *filters.Status)
		paramCount++
	}

	if filters.IsPublic != nil {
		conditions = append(conditions, fmt.Sprintf("is_public = $%d", paramCount))
		args = append(args, *filters.IsPublic)
		paramCount++
	}

	if filters.IsBeta != nil {
		conditions = append(conditions, fmt.Sprintf("is_beta = $%d", paramCount))
		args = append(args, *filters.IsBeta)
		paramCount++
	}

	return conditions, args
}

func (r *moduleRepository) buildOrderBy(filters registry.ModuleFilters) string {
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)
}

func (r *moduleRepository) buildPagination(filters registry.ModuleFilters) string {
	limit := 50
	if filters.Limit > 0 {
		limit = filters.Limit
	}

	offset := 0
	if filters.Offset > 0 {
		offset = filters.Offset
	}

	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

func boolPtr(b bool) *bool {
	return &b
}

func statusPtr(s registry.ModuleStatus) *registry.ModuleStatus {
	return &s
}
