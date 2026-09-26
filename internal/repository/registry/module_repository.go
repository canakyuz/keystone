package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/canakyuz/keystone/internal/domain/registry"
)

// ModuleRepository reads and writes the module catalogue.
type ModuleRepository struct {
	db *sql.DB
}

// NewModuleRepository creates a new module repository
func NewModuleRepository(db *sql.DB) *ModuleRepository {
	return &ModuleRepository{db: db}
}

// GetByID retrieves a module by ID
func (r *ModuleRepository) GetByID(ctx context.Context, id string) (*registry.Module, error) {
	if !isUUID(id) {
		return nil, registry.ErrModuleNotFound
	}
	return r.getOne(ctx, "id", id)
}

// GetBySlug retrieves a module by slug
func (r *ModuleRepository) GetBySlug(ctx context.Context, slug string) (*registry.Module, error) {
	return r.getOne(ctx, "slug", slug)
}

// getOne reads the live module whose column equals value. The column is one of the
// literals passed above and never comes from a request.
func (r *ModuleRepository) getOne(ctx context.Context, column, value string) (*registry.Module, error) {
	query := "SELECT " + moduleColumns + " FROM modules WHERE " + column + " = $1 AND deleted_at IS NULL"

	module, err := scanModule(r.db.QueryRowContext(ctx, query, value))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, registry.ErrModuleNotFound
	}
	return module, err
}

// List retrieves modules with filters
func (r *ModuleRepository) List(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	query := "SELECT " + moduleColumns + " FROM modules WHERE deleted_at IS NULL"

	conditions, args := r.buildFilterConditions(filters)
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += catalogOrderBy(filters.SortBy, filters.SortOrder)
	query += r.buildPagination(filters)

	return r.queryModules(ctx, query, args...)
}

// ListPublic retrieves public modules
func (r *ModuleRepository) ListPublic(ctx context.Context, filters registry.ModuleFilters) ([]*registry.Module, error) {
	filters.IsPublic = boolPtr(true)
	filters.Status = statusPtr(registry.ModuleStatusActive)
	return r.List(ctx, filters)
}

// Search searches modules
func (r *ModuleRepository) Search(ctx context.Context, query string, filters registry.ModuleFilters) ([]*registry.Module, error) {
	// Simple implementation - can be enhanced with full-text search
	searchQuery := "SELECT " + moduleColumns + ` FROM modules
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

	return r.queryModules(ctx, searchQuery, allArgs...)
}

// queryModules runs a query selecting moduleColumns and decodes every row.
func (r *ModuleRepository) queryModules(ctx context.Context, query string, args ...any) ([]*registry.Module, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []*registry.Module
	for rows.Next() {
		module, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, rows.Err()
}

// Helper functions
func (r *ModuleRepository) buildFilterConditions(filters registry.ModuleFilters) ([]string, []any) {
	return r.buildFilterConditionsWithOffset(filters, 1)
}

func (r *ModuleRepository) buildFilterConditionsWithOffset(filters registry.ModuleFilters, startParam int) ([]string, []any) {
	var conditions []string
	var args []any
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

func (r *ModuleRepository) buildPagination(filters registry.ModuleFilters) string {
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
