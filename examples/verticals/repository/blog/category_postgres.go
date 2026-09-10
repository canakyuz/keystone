package blog

import (
	"context"
	"database/sql"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/blog"
)

type CategoryPostgresRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) blog.CategoryRepository {
	return &CategoryPostgresRepository{db: db}
}

func (r *CategoryPostgresRepository) Create(ctx context.Context, category *blog.Category) error {
	query := `INSERT INTO blog_categories (id, tenant_id, name, slug, description, post_count, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, category.ID, category.TenantID, category.Name, category.Slug, category.Description, category.PostCount, category.Metadata, category.CreatedAt, category.UpdatedAt)
	return err
}

func (r *CategoryPostgresRepository) GetByID(ctx context.Context, id string) (*blog.Category, error) {
	query := `SELECT id, tenant_id, name, slug, description, post_count, metadata, created_at, updated_at FROM blog_categories WHERE id = $1 AND deleted_at IS NULL`
	category := &blog.Category{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&category.ID, &category.TenantID, &category.Name, &category.Slug, &category.Description, &category.PostCount, &category.Metadata, &category.CreatedAt, &category.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, blog.ErrCategoryNotFound
	}
	return category, err
}

func (r *CategoryPostgresRepository) GetBySlug(ctx context.Context, slug string) (*blog.Category, error) {
	query := `SELECT id, tenant_id, name, slug, description, post_count, metadata, created_at, updated_at FROM blog_categories WHERE slug = $1 AND deleted_at IS NULL`
	category := &blog.Category{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&category.ID, &category.TenantID, &category.Name, &category.Slug, &category.Description, &category.PostCount, &category.Metadata, &category.CreatedAt, &category.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, blog.ErrCategoryNotFound
	}
	return category, err
}

func (r *CategoryPostgresRepository) List(ctx context.Context, limit, offset int) ([]*blog.Category, int64, error) {
	query := `SELECT id, tenant_id, name, slug, description, post_count, metadata, created_at, updated_at FROM blog_categories WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var categories []*blog.Category
	for rows.Next() {
		category := &blog.Category{}
		if err := rows.Scan(&category.ID, &category.TenantID, &category.Name, &category.Slug, &category.Description, &category.PostCount, &category.Metadata, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, 0, err
		}
		categories = append(categories, category)
	}

	countQuery := `SELECT COUNT(*) FROM blog_categories WHERE deleted_at IS NULL`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	return categories, total, rows.Err()
}

func (r *CategoryPostgresRepository) Update(ctx context.Context, category *blog.Category) error {
	query := `UPDATE blog_categories SET name = $1, slug = $2, description = $3, post_count = $4, metadata = $5, updated_at = $6 WHERE id = $7 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, category.Name, category.Slug, category.Description, category.PostCount, category.Metadata, category.UpdatedAt, category.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return blog.ErrCategoryNotFound
	}
	return nil
}

func (r *CategoryPostgresRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE blog_categories SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return blog.ErrCategoryNotFound
	}
	return nil
}

func (r *CategoryPostgresRepository) IncrementPostCount(ctx context.Context, id string) error {
	query := `UPDATE blog_categories SET post_count = post_count + 1 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *CategoryPostgresRepository) DecrementPostCount(ctx context.Context, id string) error {
	query := `UPDATE blog_categories SET post_count = GREATEST(post_count - 1, 0) WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
