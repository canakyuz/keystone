package project

import "context"

// Repository defines the interface for project data access
type Repository interface {
	// Create creates a new project
	Create(ctx context.Context, project *Project) error

	// GetByID retrieves a project by ID
	GetByID(ctx context.Context, id string) (*Project, error)

	// GetBySlug retrieves a project by slug
	GetBySlug(ctx context.Context, slug string) (*Project, error)

	// List retrieves all projects with filters
	List(ctx context.Context, filters ListFilters) ([]*Project, int64, error)

	// Update updates an existing project
	Update(ctx context.Context, project *Project) error

	// Delete soft deletes a project
	Delete(ctx context.Context, id string) error

	// ExistsBySlug checks if a project with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// IncrementViewCount increments project view count
	IncrementViewCount(ctx context.Context, id string) error

	// GetFeatured retrieves featured projects
	GetFeatured(ctx context.Context, limit int) ([]*Project, error)

	// GetByCategory retrieves projects by category
	GetByCategory(ctx context.Context, category ProjectCategory, limit, offset int) ([]*Project, int64, error)

	// GetStats returns project statistics
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// ListFilters defines filters for listing projects
type ListFilters struct {
	Status    *ProjectStatus
	Category  *ProjectCategory
	Featured  *bool
	Search    string // Search in title, description, client
	SortBy    string // created_at, updated_at, title, view_count
	SortOrder string // asc, desc
	Limit     int
	Offset    int
}
