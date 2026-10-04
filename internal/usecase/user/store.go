package user

import (
	"context"

	"github.com/canakyuz/keystone/internal/domain/user"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
)

// Store is what the user service needs from the user repository. It is declared here, by
// its consumer, and names only the methods the service calls.
type Store interface {
	CountByRole(ctx context.Context, tenantID string, role user.UserRole) (int64, error)
	CountByStatus(ctx context.Context, tenantID string, status user.UserStatus) (int64, error)
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
	CreateAudited(ctx context.Context, u *user.User, entry auditrepo.Entry) error
	DeleteAudited(ctx context.Context, tenantID, userID string, entry auditrepo.Entry) error
	ExistsByEmail(ctx context.Context, tenantID, email string) (bool, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*user.User, error)
	GetByEmailGlobal(ctx context.Context, email string) (*user.User, error)
	GetByID(ctx context.Context, tenantID, userID string) (*user.User, error)
	List(ctx context.Context, tenantID string, filters userRepo.ListFilters) ([]*user.User, int64, error)
	Update(ctx context.Context, u *user.User) error
	UpdateAudited(ctx context.Context, u *user.User, entry auditrepo.Entry) error
	UpdateLastLogin(ctx context.Context, tenantID, userID string) error
}
