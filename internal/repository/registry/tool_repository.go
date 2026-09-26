package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// ToolRepository reads and writes the tool catalogue.
type ToolRepository struct {
	db *sql.DB
}

// NewToolRepository creates a new tool repository
func NewToolRepository(db *sql.DB) *ToolRepository {
	return &ToolRepository{db: db}
}

// GetByID retrieves a tool by ID
func (r *ToolRepository) GetByID(ctx context.Context, id string) (*registry.Tool, error) {
	if !isUUID(id) {
		return nil, registry.ErrToolNotFound
	}
	return r.getOne(ctx, "id", id)
}

// GetBySlug retrieves a tool by slug
func (r *ToolRepository) GetBySlug(ctx context.Context, slug string) (*registry.Tool, error) {
	return r.getOne(ctx, "slug", slug)
}

// getOne reads the live tool whose column equals value. The column is one of the literals
// passed above and never comes from a request.
func (r *ToolRepository) getOne(ctx context.Context, column, value string) (*registry.Tool, error) {
	query := "SELECT " + toolColumns + " FROM tools WHERE " + column + " = $1 AND deleted_at IS NULL"

	tool, err := scanTool(r.db.QueryRowContext(ctx, query, value))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, registry.ErrToolNotFound
	}
	return tool, err
}

// List retrieves tools with filters
func (r *ToolRepository) List(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	query := "SELECT " + toolColumns + " FROM tools WHERE deleted_at IS NULL"

	conditions, args := r.buildFilterConditions(filters)
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += catalogOrderBy(filters.SortBy, filters.SortOrder)
	query += r.buildPagination(filters)

	return r.queryTools(ctx, query, args...)
}

// ListPublic retrieves public tools
func (r *ToolRepository) ListPublic(ctx context.Context, filters registry.ToolFilters) ([]*registry.Tool, error) {
	filters.IsPublic = boolPtr(true)
	filters.Status = toolStatusPtr(registry.ToolStatusActive)
	return r.List(ctx, filters)
}

// Search searches tools
func (r *ToolRepository) Search(ctx context.Context, query string, filters registry.ToolFilters) ([]*registry.Tool, error) {
	searchQuery := "SELECT " + toolColumns + ` FROM tools
		WHERE deleted_at IS NULL
		  AND (name ILIKE $1 OR description ILIKE $1 OR ARRAY_TO_STRING(tags, ' ') ILIKE $1)`

	// $1 is used for search pattern, so filter conditions start at $2
	conditions, args := r.buildFilterConditionsWithOffset(filters, 2)
	searchPattern := "%" + query + "%"
	allArgs := append([]any{searchPattern}, args...)

	if len(conditions) > 0 {
		searchQuery += " AND " + strings.Join(conditions, " AND ")
	}

	searchQuery += catalogOrderBy(filters.SortBy, filters.SortOrder)
	searchQuery += r.buildPagination(filters)

	return r.queryTools(ctx, searchQuery, allArgs...)
}

// queryTools runs a query selecting toolColumns and decodes every row.
func (r *ToolRepository) queryTools(ctx context.Context, query string, args ...any) ([]*registry.Tool, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []*registry.Tool
	for rows.Next() {
		tool, err := scanTool(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}

	return tools, rows.Err()
}

// Helper functions
func (r *ToolRepository) buildFilterConditions(filters registry.ToolFilters) ([]string, []any) {
	return r.buildFilterConditionsWithOffset(filters, 1)
}

func (r *ToolRepository) buildFilterConditionsWithOffset(filters registry.ToolFilters, startParam int) ([]string, []any) {
	var conditions []string
	var args []any
	paramCount := startParam

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

func (r *ToolRepository) buildPagination(filters registry.ToolFilters) string {
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
