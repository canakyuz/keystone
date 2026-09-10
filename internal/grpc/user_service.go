package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainUser "github.com/canakyuz/keystone/internal/domain/user"
	keystonev1 "github.com/canakyuz/keystone/internal/grpc/keystone/v1"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	"github.com/canakyuz/keystone/pkg/tenantctx"
)

// maxPageSize caps what a caller can ask for.
//
// Without a cap, limit is a caller-controlled multiplier on the work the server does and
// the memory it holds, which is a denial of service with a legitimate-looking request.
const maxPageSize = 200

// UserStore is the storage behaviour this service needs.
type UserStore interface {
	List(ctx context.Context, tenantID string, filters userRepo.ListFilters) ([]*domainUser.User, int64, error)
}

// UserService serves keystone.v1.UserService.
type UserService struct {
	keystonev1.UnimplementedUserServiceServer

	store UserStore
}

// NewUserService builds the service.
func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

// ListUsers returns users belonging to the caller's tenant.
//
// The tenant comes from the context the auth interceptor populated, never from the
// request. There is no tenant field on ListUsersRequest for the same reason: a field the
// caller controls is a field the caller can change, and the isolation guarantee is
// precisely that they cannot.
func (s *UserService) ListUsers(
	ctx context.Context, req *keystonev1.ListUsersRequest,
) (*keystonev1.ListUsersResponse, error) {
	tenantID := tenantctx.ID(ctx)
	if tenantID == "" {
		// Unreachable through the interceptor chain, which rejects a call with no tenant
		// before it gets here. Kept because "unreachable" is a property of today's wiring,
		// and the failure if the wiring changes would be an unscoped query.
		return nil, status.Error(codes.Unauthenticated, "no tenant in context")
	}

	limit := int(req.GetLimit())
	if limit <= 0 || limit > maxPageSize {
		limit = maxPageSize
	}

	offset := int(req.GetOffset())
	if offset < 0 {
		offset = 0
	}

	users, total, err := s.store.List(ctx, tenantID, userRepo.ListFilters{Limit: limit, Offset: offset})
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list users")
	}

	out := make([]*keystonev1.User, 0, len(users))
	for _, u := range users {
		out = append(out, &keystonev1.User{
			Id:        u.ID,
			TenantId:  u.TenantID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Role:      string(u.Role),
			Status:    string(u.Status),
		})
	}

	return &keystonev1.ListUsersResponse{Users: out, Total: total}, nil
}
