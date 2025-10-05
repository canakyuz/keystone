package registry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"nexpaces-api/internal/domain/registry"
)

type toolRepository struct {
	db *sql.DB
}

// NewToolRepository creates a new tool repository
func NewToolRepository(db *sql.DB) registry.ToolRepository {
	return &toolRepository{db: db}
}

// Create creates a new tool
func (r *toolRepository) Create(ctx context.Context, tool *registry.Tool) error {
	query := `
		INSERT INTO tools (
			id, name, slug, code, display_name, description,
			category, tool_type, scope, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			transaction_fee_percentage, transaction_fee_fixed,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			integration_type, provider_slug, provider_name, requires_api_keys,
			api_credentials_schema, webhook_config, supported_triggers,
			default_config, default_limits, installation_notes, configuration_schema,
			supported_events, code_examples, sdk_info, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34,
			$35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		tool.ID, tool.Name, tool.Slug, tool.Code, tool.DisplayName, tool.Description,
		tool.Category, tool.ToolType, tool.Scope, tool.Status, tool.IsPublic, tool.IsBeta,
		tool.Version, tool.MinPlatformVersion, tool.PricingModel, tool.BasePrice, tool.Currency, tool.BillingCycle,
		tool.TransactionFeePercentage, tool.TransactionFeeFixed,
		tool.Features, tool.Capabilities, tool.Icon, tool.CoverImage, tool.Screenshots, tool.DemoURL, tool.DocumentationURL,
		tool.IntegrationType, tool.ProviderSlug, tool.ProviderName, tool.RequiresAPIKeys,
		tool.APICredentialsSchema, tool.WebhookConfig, tool.SupportedTriggers,
		tool.DefaultConfig, tool.DefaultLimits, tool.InstallationNotes, tool.ConfigurationSchema,
		tool.SupportedEvents, tool.CodeExamples, tool.SDKInfo, tool.Tags, tool.Metadata,
		tool.InstallCount, tool.Rating, tool.ReviewCount,
		tool.CreatedAt, tool.UpdatedAt, tool.CreatedBy, tool.UpdatedBy,
	)

	return err
}

// GetByID retrieves a tool by ID
func (r *toolRepository) GetByID(ctx context.Context, id string) (*registry.Tool, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, tool_type, scope, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			transaction_fee_percentage, transaction_fee_fixed,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			integration_type, provider_slug, provider_name, requires_api_keys,
			api_credentials_schema, webhook_config, supported_triggers,
			default_config, default_limits, installation_notes, configuration_schema,
			supported_events, code_examples, sdk_info, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM tools
		WHERE id = $1 AND deleted_at IS NULL
	`

	tool := &registry.Tool{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tool.ID, &tool.Name, &tool.Slug, &tool.Code, &tool.DisplayName, &tool.Description,
		&tool.Category, &tool.ToolType, &tool.Scope, &tool.Status, &tool.IsPublic, &tool.IsBeta,
		&tool.Version, &tool.MinPlatformVersion, &tool.PricingModel, &tool.BasePrice, &tool.Currency, &tool.BillingCycle,
		&tool.TransactionFeePercentage, &tool.TransactionFeeFixed,
		&tool.Features, &tool.Capabilities, &tool.Icon, &tool.CoverImage, &tool.Screenshots, &tool.DemoURL, &tool.DocumentationURL,
		&tool.IntegrationType, &tool.ProviderSlug, &tool.ProviderName, &tool.RequiresAPIKeys,
		&tool.APICredentialsSchema, &tool.WebhookConfig, &tool.SupportedTriggers,
		&tool.DefaultConfig, &tool.DefaultLimits, &tool.InstallationNotes, &tool.ConfigurationSchema,
		&tool.SupportedEvents, &tool.CodeExamples, &tool.SDKInfo, &tool.Tags, &tool.Metadata,
		&tool.InstallCount, &tool.Rating, &tool.ReviewCount,
		&tool.CreatedAt, &tool.UpdatedAt, &tool.DeletedAt, &tool.CreatedBy, &tool.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrToolNotFound
	}

	return tool, err
}

// GetByCode retrieves a tool by code
func (r *toolRepository) GetByCode(ctx context.Context, code string) (*registry.Tool, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, tool_type, scope, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			transaction_fee_percentage, transaction_fee_fixed,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			integration_type, provider_slug, provider_name, requires_api_keys,
			api_credentials_schema, webhook_config, supported_triggers,
			default_config, default_limits, installation_notes, configuration_schema,
			supported_events, code_examples, sdk_info, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM tools
		WHERE code = $1 AND deleted_at IS NULL
	`

	tool := &registry.Tool{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&tool.ID, &tool.Name, &tool.Slug, &tool.Code, &tool.DisplayName, &tool.Description,
		&tool.Category, &tool.ToolType, &tool.Scope, &tool.Status, &tool.IsPublic, &tool.IsBeta,
		&tool.Version, &tool.MinPlatformVersion, &tool.PricingModel, &tool.BasePrice, &tool.Currency, &tool.BillingCycle,
		&tool.TransactionFeePercentage, &tool.TransactionFeeFixed,
		&tool.Features, &tool.Capabilities, &tool.Icon, &tool.CoverImage, &tool.Screenshots, &tool.DemoURL, &tool.DocumentationURL,
		&tool.IntegrationType, &tool.ProviderSlug, &tool.ProviderName, &tool.RequiresAPIKeys,
		&tool.APICredentialsSchema, &tool.WebhookConfig, &tool.SupportedTriggers,
		&tool.DefaultConfig, &tool.DefaultLimits, &tool.InstallationNotes, &tool.ConfigurationSchema,
		&tool.SupportedEvents, &tool.CodeExamples, &tool.SDKInfo, &tool.Tags, &tool.Metadata,
		&tool.InstallCount, &tool.Rating, &tool.ReviewCount,
		&tool.CreatedAt, &tool.UpdatedAt, &tool.DeletedAt, &tool.CreatedBy, &tool.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrToolNotFound
	}

	return tool, err
}

// GetBySlug retrieves a tool by slug
func (r *toolRepository) GetBySlug(ctx context.Context, slug string) (*registry.Tool, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, tool_type, scope, status, is_public, is_beta,
			version, min_platform_version, pricing_model, base_price, currency, billing_cycle,
			transaction_fee_percentage, transaction_fee_fixed,
			features, capabilities, icon, cover_image, screenshots, demo_url, documentation_url,
			integration_type, provider_slug, provider_name, requires_api_keys,
			api_credentials_schema, webhook_config, supported_triggers,
			default_config, default_limits, installation_notes, configuration_schema,
			supported_events, code_examples, sdk_info, tags, metadata,
			install_count, rating, review_count,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM tools
		WHERE slug = $1 AND deleted_at IS NULL
	`

	tool := &registry.Tool{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tool.ID, &tool.Name, &tool.Slug, &tool.Code, &tool.DisplayName, &tool.Description,
		&tool.Category, &tool.ToolType, &tool.Scope, &tool.Status, &tool.IsPublic, &tool.IsBeta,
		&tool.Version, &tool.MinPlatformVersion, &tool.PricingModel, &tool.BasePrice, &tool.Currency, &tool.BillingCycle,
		&tool.TransactionFeePercentage, &tool.TransactionFeeFixed,
		&tool.Features, &tool.Capabilities, &tool.Icon, &tool.CoverImage, &tool.Screenshots, &tool.DemoURL, &tool.DocumentationURL,
		&tool.IntegrationType, &tool.ProviderSlug, &tool.ProviderName, &tool.RequiresAPIKeys,
		&tool.APICredentialsSchema, &tool.WebhookConfig, &tool.SupportedTriggers,
		&tool.DefaultConfig, &tool.DefaultLimits, &tool.InstallationNotes, &tool.ConfigurationSchema,
		&tool.SupportedEvents, &tool.CodeExamples, &tool.SDKInfo, &tool.Tags, &tool.Metadata,
		&tool.InstallCount, &tool.Rating, &tool.ReviewCount,
		&tool.CreatedAt, &tool.UpdatedAt, &tool.DeletedAt, &tool.CreatedBy, &tool.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, registry.ErrToolNotFound
	}

	return tool, err
}

// Update updates a tool
func (r *toolRepository) Update(ctx context.Context, tool *registry.Tool) error {
	query := `
		UPDATE tools SET
			name = $2, slug = $3, display_name = $4, description = $5,
			category = $6, tool_type = $7, scope = $8, status = $9, is_public = $10, is_beta = $11,
			version = $12, pricing_model = $13, base_price = $14, billing_cycle = $15,
			transaction_fee_percentage = $16, transaction_fee_fixed = $17,
			features = $18, capabilities = $19, icon = $20, cover_image = $21,
			provider_name = $22, default_config = $23, default_limits = $24,
			tags = $25, metadata = $26, rating = $27,
			updated_at = $28, updated_by = $29
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query,
		tool.ID, tool.Name, tool.Slug, tool.DisplayName, tool.Description,
		tool.Category, tool.ToolType, tool.Scope, tool.Status, tool.IsPublic, tool.IsBeta,
		tool.Version, tool.PricingModel, tool.BasePrice, tool.BillingCycle,
		tool.TransactionFeePercentage, tool.TransactionFeeFixed,
		tool.Features, tool.Capabilities, tool.Icon, tool.CoverImage,
		tool.ProviderName, tool.DefaultConfig, tool.DefaultLimits,
		tool.Tags, tool.Metadata, tool.Rating,
		tool.UpdatedAt, tool.UpdatedBy,
	)

	return err
}

// Delete soft deletes a tool
func (r *toolRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE tools SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List retrieves tools with filters
func (r *toolRepository) List(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	query := `
		SELECT id, name, slug, code, display_name, description,
			category, tool_type, scope, status, is_public, is_beta,
			version, pricing_model, base_price, currency,
			transaction_fee_percentage, transaction_fee_fixed,
			icon, install_count, rating, review_count,
			created_at, updated_at
		FROM tools
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

	var tools []*registry.Tool
	for rows.Next() {
		tool := &registry.Tool{}
		err := rows.Scan(
			&tool.ID, &tool.Name, &tool.Slug, &tool.Code, &tool.DisplayName, &tool.Description,
			&tool.Category, &tool.ToolType, &tool.Scope, &tool.Status, &tool.IsPublic, &tool.IsBeta,
			&tool.Version, &tool.PricingModel, &tool.BasePrice, &tool.Currency,
			&tool.TransactionFeePercentage, &tool.TransactionFeeFixed,
			&tool.Icon, &tool.InstallCount, &tool.Rating, &tool.ReviewCount,
			&tool.CreatedAt, &tool.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}

	return tools, rows.Err()
}

// ListPublic retrieves public tools
func (r *toolRepository) ListPublic(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)
	return r.List(ctx, filters)
}

// Search searches tools
func (r *toolRepository) Search(ctx context.Context, query string, filters registry.ToolFilters) ([]*registry.Tool, error) {
	searchQuery := `
		SELECT id, name, slug, code, display_name, description,
			category, tool_type, scope, status, is_public, is_beta,
			version, pricing_model, base_price, currency,
			transaction_fee_percentage, transaction_fee_fixed,
			icon, install_count, rating, review_count,
			created_at, updated_at
		FROM tools
		WHERE deleted_at IS NULL
		  AND (name ILIKE $1 OR description ILIKE $1 OR ARRAY_TO_STRING(tags, ' ') ILIKE $1)
	`

	conditions, args := r.buildFilterConditions(filters)
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

	var tools []*registry.Tool
	for rows.Next() {
		tool := &registry.Tool{}
		err := rows.Scan(
			&tool.ID, &tool.Name, &tool.Slug, &tool.Code, &tool.DisplayName, &tool.Description,
			&tool.Category, &tool.ToolType, &tool.Scope, &tool.Status, &tool.IsPublic, &tool.IsBeta,
			&tool.Version, &tool.PricingModel, &tool.BasePrice, &tool.Currency,
			&tool.TransactionFeePercentage, &tool.TransactionFeeFixed,
			&tool.Icon, &tool.InstallCount, &tool.Rating, &tool.ReviewCount,
			&tool.CreatedAt, &tool.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}

	return tools, rows.Err()
}

// GetInstallCount retrieves install count
func (r *toolRepository) GetInstallCount(ctx context.Context, toolID string) (int, error) {
	var count int
	query := `SELECT install_count FROM tools WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, toolID).Scan(&count)
	return count, err
}

// UpdateInstallCount updates install count
func (r *toolRepository) UpdateInstallCount(ctx context.Context, toolID string, delta int) error {
	query := `UPDATE tools SET install_count = install_count + $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, toolID, delta)
	return err
}

// Helper functions
func (r *toolRepository) buildFilterConditions(filters registry.ToolFilters) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	paramCount := 1

	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", paramCount))
		args = append(args, *filters.Category)
		paramCount++
	}

	if filters.ToolType != nil {
		conditions = append(conditions, fmt.Sprintf("tool_type = $%d", paramCount))
		args = append(args, *filters.ToolType)
		paramCount++
	}

	if filters.Scope != nil {
		conditions = append(conditions, fmt.Sprintf("scope = $%d", paramCount))
		args = append(args, *filters.Scope)
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

func (r *toolRepository) buildOrderBy(filters registry.ToolFilters) string {
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

func (r *toolRepository) buildPagination(filters registry.ToolFilters) string {
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

func toolStatusPtr(s registry.ToolStatus) *registry.ToolStatus {
	return &s
}
