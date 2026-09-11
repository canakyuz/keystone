package blog

import "context"

type PostRepository interface {
	Create(ctx context.Context, post *Post) error
	GetByID(ctx context.Context, id string) (*Post, error)
	GetBySlug(ctx context.Context, slug string) (*Post, error)
	List(ctx context.Context, filters PostListFilters) ([]*Post, int64, error)
	GetFeatured(ctx context.Context, limit int) ([]*Post, error)
	GetByCategory(ctx context.Context, categoryID string, limit, offset int) ([]*Post, int64, error)
	GetByTag(ctx context.Context, tag string, limit, offset int) ([]*Post, int64, error)
	Update(ctx context.Context, post *Post) error
	Delete(ctx context.Context, id string) error
	IncrementViewCount(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]any, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, category *Category) error
	GetByID(ctx context.Context, id string) (*Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	List(ctx context.Context, limit, offset int) ([]*Category, int64, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id string) error
}

type PostListFilters struct {
	Status     *PostStatus
	CategoryID *string
	Tag        *string
	Featured   *bool
	Search     string
	SortBy     string
	SortOrder  string
	Limit      int
	Offset     int
}
