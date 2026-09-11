package blog

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/blog"
	"github.com/lib/pq"
)

type PostPostgresRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) blog.PostRepository {
	return &PostPostgresRepository{db: db}
}

func (r *PostPostgresRepository) Create(ctx context.Context, post *blog.Post) error {
	query := `INSERT INTO blog_posts (id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	_, err := r.db.ExecContext(ctx, query, post.ID, post.TenantID, post.CategoryID, post.Title, post.Slug, post.Content, post.Excerpt, post.Status, post.Featured, post.ViewCount, post.Image, pq.Array(post.Tags), post.PublishedAt, post.Metadata, post.CreatedAt, post.UpdatedAt, post.CreatedBy, post.UpdatedBy)
	return err
}

func (r *PostPostgresRepository) GetByID(ctx context.Context, id string) (*blog.Post, error) {
	query := `SELECT id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by FROM blog_posts WHERE id = $1 AND deleted_at IS NULL`
	post := &blog.Post{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&post.ID, &post.TenantID, &post.CategoryID, &post.Title, &post.Slug, &post.Content, &post.Excerpt, &post.Status, &post.Featured, &post.ViewCount, &post.Image, pq.Array(&post.Tags), &post.PublishedAt, &post.Metadata, &post.CreatedAt, &post.UpdatedAt, &post.CreatedBy, &post.UpdatedBy)
	if err == sql.ErrNoRows {
		return nil, blog.ErrPostNotFound
	}
	return post, err
}

func (r *PostPostgresRepository) GetBySlug(ctx context.Context, slug string) (*blog.Post, error) {
	query := `SELECT id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by FROM blog_posts WHERE slug = $1 AND deleted_at IS NULL`
	post := &blog.Post{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&post.ID, &post.TenantID, &post.CategoryID, &post.Title, &post.Slug, &post.Content, &post.Excerpt, &post.Status, &post.Featured, &post.ViewCount, &post.Image, pq.Array(&post.Tags), &post.PublishedAt, &post.Metadata, &post.CreatedAt, &post.UpdatedAt, &post.CreatedBy, &post.UpdatedBy)
	if err == sql.ErrNoRows {
		return nil, blog.ErrPostNotFound
	}
	return post, err
}

func (r *PostPostgresRepository) List(ctx context.Context, filters blog.PostListFilters) ([]*blog.Post, int64, error) {
	query := `SELECT id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by FROM blog_posts WHERE deleted_at IS NULL`
	var args []any
	argPos := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filters.Status)
		argPos++
	}
	if filters.CategoryID != nil {
		query += fmt.Sprintf(" AND category_id = $%d", argPos)
		args = append(args, *filters.CategoryID)
		argPos++
	}
	if filters.Featured != nil {
		query += fmt.Sprintf(" AND featured = $%d", argPos)
		args = append(args, *filters.Featured)
		argPos++
	}

	countQuery := `SELECT COUNT(*) FROM (` + query + `) AS count_table`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*blog.Post
	for rows.Next() {
		post := &blog.Post{}
		if err := rows.Scan(&post.ID, &post.TenantID, &post.CategoryID, &post.Title, &post.Slug, &post.Content, &post.Excerpt, &post.Status, &post.Featured, &post.ViewCount, &post.Image, pq.Array(&post.Tags), &post.PublishedAt, &post.Metadata, &post.CreatedAt, &post.UpdatedAt, &post.CreatedBy, &post.UpdatedBy); err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	return posts, total, rows.Err()
}

func (r *PostPostgresRepository) Update(ctx context.Context, post *blog.Post) error {
	query := `UPDATE blog_posts SET category_id = $1, title = $2, slug = $3, content = $4, excerpt = $5, status = $6, featured = $7, image = $8, tags = $9, published_at = $10, metadata = $11, updated_at = $12, updated_by = $13 WHERE id = $14 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, post.CategoryID, post.Title, post.Slug, post.Content, post.Excerpt, post.Status, post.Featured, post.Image, pq.Array(post.Tags), post.PublishedAt, post.Metadata, post.UpdatedAt, post.UpdatedBy, post.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return blog.ErrPostNotFound
	}
	return nil
}

func (r *PostPostgresRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE blog_posts SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return blog.ErrPostNotFound
	}
	return nil
}

func (r *PostPostgresRepository) GetByTag(ctx context.Context, tag string, limit, offset int) ([]*blog.Post, int64, error) {
	query := `SELECT id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by FROM blog_posts WHERE $1 = ANY(tags) AND deleted_at IS NULL ORDER BY published_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, tag, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*blog.Post
	for rows.Next() {
		post := &blog.Post{}
		if err := rows.Scan(&post.ID, &post.TenantID, &post.CategoryID, &post.Title, &post.Slug, &post.Content, &post.Excerpt, &post.Status, &post.Featured, &post.ViewCount, &post.Image, pq.Array(&post.Tags), &post.PublishedAt, &post.Metadata, &post.CreatedAt, &post.UpdatedAt, &post.CreatedBy, &post.UpdatedBy); err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE $1 = ANY(tags) AND deleted_at IS NULL`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, tag).Scan(&total); err != nil {
		return nil, 0, err
	}

	return posts, total, rows.Err()
}

func (r *PostPostgresRepository) GetFeatured(ctx context.Context, limit int) ([]*blog.Post, error) {
	query := `SELECT id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by FROM blog_posts WHERE featured = true AND status = 'published' AND deleted_at IS NULL ORDER BY published_at DESC LIMIT $1`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*blog.Post
	for rows.Next() {
		post := &blog.Post{}
		if err := rows.Scan(&post.ID, &post.TenantID, &post.CategoryID, &post.Title, &post.Slug, &post.Content, &post.Excerpt, &post.Status, &post.Featured, &post.ViewCount, &post.Image, pq.Array(&post.Tags), &post.PublishedAt, &post.Metadata, &post.CreatedAt, &post.UpdatedAt, &post.CreatedBy, &post.UpdatedBy); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

func (r *PostPostgresRepository) IncrementViewCount(ctx context.Context, id string) error {
	query := `UPDATE blog_posts SET view_count = view_count + 1 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostPostgresRepository) GetByCategory(ctx context.Context, categoryID string, limit, offset int) ([]*blog.Post, int64, error) {
	query := `SELECT id, tenant_id, category_id, title, slug, content, excerpt, status, featured, view_count, image, tags, published_at, metadata, created_at, updated_at, created_by, updated_by FROM blog_posts WHERE category_id = $1 AND deleted_at IS NULL ORDER BY published_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, categoryID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*blog.Post
	for rows.Next() {
		post := &blog.Post{}
		if err := rows.Scan(&post.ID, &post.TenantID, &post.CategoryID, &post.Title, &post.Slug, &post.Content, &post.Excerpt, &post.Status, &post.Featured, &post.ViewCount, &post.Image, pq.Array(&post.Tags), &post.PublishedAt, &post.Metadata, &post.CreatedAt, &post.UpdatedAt, &post.CreatedBy, &post.UpdatedBy); err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE category_id = $1 AND deleted_at IS NULL`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, categoryID).Scan(&total); err != nil {
		return nil, 0, err
	}

	return posts, total, rows.Err()
}

func (r *PostPostgresRepository) GetStats(ctx context.Context) (map[string]any, error) {
	stats := make(map[string]any)

	// Total posts
	var totalPosts int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM blog_posts WHERE deleted_at IS NULL`).Scan(&totalPosts)
	if err != nil {
		return nil, err
	}
	stats["total_posts"] = totalPosts

	// Published posts
	var publishedPosts int64
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM blog_posts WHERE status = 'published' AND deleted_at IS NULL`).Scan(&publishedPosts)
	if err != nil {
		return nil, err
	}
	stats["published_posts"] = publishedPosts

	// Draft posts
	var draftPosts int64
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM blog_posts WHERE status = 'draft' AND deleted_at IS NULL`).Scan(&draftPosts)
	if err != nil {
		return nil, err
	}
	stats["draft_posts"] = draftPosts

	return stats, nil
}
