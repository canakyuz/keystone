package user

import (
	"github.com/canakyuz/keystone/internal/domain/user"
)

// ListFilters represents filters for listing users
type ListFilters struct {
	// Pagination
	Limit  int
	Offset int

	// Filters
	Role   *user.UserRole
	Status *user.UserStatus
	Search string // Search in email, first_name, last_name

	// Sorting
	SortBy    string // created_at, email, first_name, last_name
	SortOrder string // asc, desc
}
