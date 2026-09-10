package project

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/canakyuz/keystone/examples/verticals/domain/project"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new project
func (r *PostgresRepository) Create(ctx context.Context, p *project.Project) error {
	query := `
		INSERT INTO projects (
			id, tenant_id, title, slug, description, content,
			category, status, client, technologies, images, cover_image,
			live_url, github_url, start_date, end_date,
			featured, sort_order, view_count, metadata,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23, $24
		)
	`

	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		p.ID, p.TenantID, p.Title, p.Slug, p.Description, p.Content,
		p.Category, p.Status, p.Client, pq.Array(p.Technologies), pq.Array(p.Images), p.CoverImage,
		p.LiveURL, p.GithubURL, p.StartDate, p.EndDate,
		p.Featured, p.SortOrder, p.ViewCount, metadataJSON,
		p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetByID retrieves a project by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*project.Project, error) {
	query := `
		SELECT
			id, tenant_id, title, slug, description, content,
			category, status, client, technologies, images, cover_image,
			live_url, github_url, start_date, end_date,
			featured, sort_order, view_count, metadata,
			created_at, updated_at, created_by, updated_by
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
	`

	p := &project.Project{}
	var metadataJSON []byte
	var technologies, images []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.TenantID, &p.Title, &p.Slug, &p.Description, &p.Content,
		&p.Category, &p.Status, &p.Client, pq.Array(&technologies), pq.Array(&images), &p.CoverImage,
		&p.LiveURL, &p.GithubURL, &p.StartDate, &p.EndDate,
		&p.Featured, &p.SortOrder, &p.ViewCount, &metadataJSON,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, project.ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	p.Technologies = technologies
	p.Images = images

	if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return p, nil
}

// GetBySlug retrieves a project by slug
func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*project.Project, error) {
	query := `
		SELECT
			id, tenant_id, title, slug, description, content,
			category, status, client, technologies, images, cover_image,
			live_url, github_url, start_date, end_date,
			featured, sort_order, view_count, metadata,
			created_at, updated_at, created_by, updated_by
		FROM projects
		WHERE slug = $1 AND deleted_at IS NULL
	`

	p := &project.Project{}
	var metadataJSON []byte
	var technologies, images []string

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&p.ID, &p.TenantID, &p.Title, &p.Slug, &p.Description, &p.Content,
		&p.Category, &p.Status, &p.Client, pq.Array(&technologies), pq.Array(&images), &p.CoverImage,
		&p.LiveURL, &p.GithubURL, &p.StartDate, &p.EndDate,
		&p.Featured, &p.SortOrder, &p.ViewCount, &metadataJSON,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, project.ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project by slug: %w", err)
	}

	p.Technologies = technologies
	p.Images = images

	if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return p, nil
}

// List retrieves all projects with filters
func (r *PostgresRepository) List(ctx context.Context, filters project.ListFilters) ([]*project.Project, int64, error) {
	whereClause, args := buildWhereClause(filters)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM projects WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
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
			id, tenant_id, title, slug, description, content,
			category, status, client, technologies, images, cover_image,
			live_url, github_url, start_date, end_date,
			featured, sort_order, view_count, metadata,
			created_at, updated_at, created_by, updated_by
		FROM projects
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, len(args)+1, len(args)+2)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]*project.Project, 0)
	for rows.Next() {
		p := &project.Project{}
		var metadataJSON []byte
		var technologies, images []string

		err := rows.Scan(
			&p.ID, &p.TenantID, &p.Title, &p.Slug, &p.Description, &p.Content,
			&p.Category, &p.Status, &p.Client, pq.Array(&technologies), pq.Array(&images), &p.CoverImage,
			&p.LiveURL, &p.GithubURL, &p.StartDate, &p.EndDate,
			&p.Featured, &p.SortOrder, &p.ViewCount, &metadataJSON,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}

		p.Technologies = technologies
		p.Images = images

		if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		projects = append(projects, p)
	}

	return projects, total, nil
}

// Update updates an existing project
func (r *PostgresRepository) Update(ctx context.Context, p *project.Project) error {
	query := `
		UPDATE projects SET
			title = $2,
			slug = $3,
			description = $4,
			content = $5,
			category = $6,
			status = $7,
			client = $8,
			technologies = $9,
			images = $10,
			cover_image = $11,
			live_url = $12,
			github_url = $13,
			start_date = $14,
			end_date = $15,
			featured = $16,
			sort_order = $17,
			view_count = $18,
			metadata = $19,
			updated_at = $20,
			updated_by = $21
		WHERE id = $1 AND deleted_at IS NULL
	`

	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	p.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		p.ID,
		p.Title, p.Slug, p.Description, p.Content,
		p.Category, p.Status, p.Client, pq.Array(p.Technologies), pq.Array(p.Images), p.CoverImage,
		p.LiveURL, p.GithubURL, p.StartDate, p.EndDate,
		p.Featured, p.SortOrder, p.ViewCount, metadataJSON,
		p.UpdatedAt, p.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return project.ErrProjectNotFound
	}

	return nil
}

// Delete soft deletes a project
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE projects
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return project.ErrProjectNotFound
	}

	return nil
}

// ExistsBySlug checks if a project with the given slug exists
func (r *PostgresRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM projects WHERE slug = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check project existence by slug: %w", err)
	}

	return exists, nil
}

// IncrementViewCount increments project view count
func (r *PostgresRepository) IncrementViewCount(ctx context.Context, id string) error {
	query := `
		UPDATE projects
		SET view_count = view_count + 1, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment view count: %w", err)
	}

	return nil
}

// GetFeatured retrieves featured projects
func (r *PostgresRepository) GetFeatured(ctx context.Context, limit int) ([]*project.Project, error) {
	query := `
		SELECT
			id, tenant_id, title, slug, description, content,
			category, status, client, technologies, images, cover_image,
			live_url, github_url, start_date, end_date,
			featured, sort_order, view_count, metadata,
			created_at, updated_at, created_by, updated_by
		FROM projects
		WHERE featured = TRUE AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured projects: %w", err)
	}
	defer rows.Close()

	projects := make([]*project.Project, 0)
	for rows.Next() {
		p := &project.Project{}
		var metadataJSON []byte
		var technologies, images []string

		err := rows.Scan(
			&p.ID, &p.TenantID, &p.Title, &p.Slug, &p.Description, &p.Content,
			&p.Category, &p.Status, &p.Client, pq.Array(&technologies), pq.Array(&images), &p.CoverImage,
			&p.LiveURL, &p.GithubURL, &p.StartDate, &p.EndDate,
			&p.Featured, &p.SortOrder, &p.ViewCount, &metadataJSON,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		p.Technologies = technologies
		p.Images = images

		if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		projects = append(projects, p)
	}

	return projects, nil
}

// GetByCategory retrieves projects by category
func (r *PostgresRepository) GetByCategory(ctx context.Context, category project.ProjectCategory, limit, offset int) ([]*project.Project, int64, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM projects WHERE category = $1 AND deleted_at IS NULL`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, category).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count projects by category: %w", err)
	}

	// List query
	query := `
		SELECT
			id, tenant_id, title, slug, description, content,
			category, status, client, technologies, images, cover_image,
			live_url, github_url, start_date, end_date,
			featured, sort_order, view_count, metadata,
			created_at, updated_at, created_by, updated_by
		FROM projects
		WHERE category = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, category, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get projects by category: %w", err)
	}
	defer rows.Close()

	projects := make([]*project.Project, 0)
	for rows.Next() {
		p := &project.Project{}
		var metadataJSON []byte
		var technologies, images []string

		err := rows.Scan(
			&p.ID, &p.TenantID, &p.Title, &p.Slug, &p.Description, &p.Content,
			&p.Category, &p.Status, &p.Client, pq.Array(&technologies), pq.Array(&images), &p.CoverImage,
			&p.LiveURL, &p.GithubURL, &p.StartDate, &p.EndDate,
			&p.Featured, &p.SortOrder, &p.ViewCount, &metadataJSON,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}

		p.Technologies = technologies
		p.Images = images

		if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		projects = append(projects, p)
	}

	return projects, total, nil
}

// GetStats returns project statistics
func (r *PostgresRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'in-progress') as in_progress,
			COUNT(*) FILTER (WHERE status = 'planned') as planned,
			COUNT(*) FILTER (WHERE featured = TRUE) as featured,
			SUM(view_count) as total_views
		FROM projects
		WHERE deleted_at IS NULL
	`

	var total, completed, inProgress, planned, featured, totalViews int64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&total, &completed, &inProgress, &planned, &featured, &totalViews,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := map[string]interface{}{
		"total":       total,
		"completed":   completed,
		"in_progress": inProgress,
		"planned":     planned,
		"featured":    featured,
		"total_views": totalViews,
	}

	return stats, nil
}

// buildWhereClause builds WHERE clause for list queries
func buildWhereClause(filters project.ListFilters) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argCount := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}

	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argCount))
		args = append(args, *filters.Category)
		argCount++
	}

	if filters.Featured != nil {
		conditions = append(conditions, fmt.Sprintf("featured = $%d", argCount))
		args = append(args, *filters.Featured)
		argCount++
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(title ILIKE $%d OR description ILIKE $%d OR client ILIKE $%d OR content ILIKE $%d)",
			argCount, argCount, argCount, argCount,
		))
		args = append(args, searchPattern)
		argCount++
	}

	return strings.Join(conditions, " AND "), args
}
