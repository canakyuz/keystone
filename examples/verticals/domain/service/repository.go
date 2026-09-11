package service

import "context"

// ServiceRepository defines service data access interface
type ServiceRepository interface {
	Create(ctx context.Context, service *Service) error
	GetByID(ctx context.Context, id string) (*Service, error)
	GetBySlug(ctx context.Context, slug string) (*Service, error)
	List(ctx context.Context, filters ServiceListFilters) ([]*Service, int64, error)
	GetFeatured(ctx context.Context, limit int) ([]*Service, error)
	GetByCategory(ctx context.Context, category ServiceCategory, limit, offset int) ([]*Service, int64, error)
	Update(ctx context.Context, service *Service) error
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]any, error)
}

// ServiceListFilters defines filters for listing services
type ServiceListFilters struct {
	Category  *ServiceCategory
	Status    *ServiceStatus
	IsPublic  *bool
	Featured  *bool
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}
