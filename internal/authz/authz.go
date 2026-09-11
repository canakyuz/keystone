// Package authz decides whether a verified subject may act inside a tenant.
//
// pkg/authn answers "is this token genuine". It cannot answer "is the subject still a
// member of the tenant the token names". A token is a statement made when it was issued,
// and it stays valid for a day after the membership behind it has been suspended, deleted
// or demoted. Only the database knows the answer now, so this package asks it.
//
// It stands apart from both transports for the reason pkg/authn does: the HTTP middleware
// and the gRPC interceptor have to reach the same verdict, and one implementation is how
// that is guaranteed.
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/canakyuz/keystone/internal/domain/user"
)

// ErrNotMember is returned when the subject holds no active membership in the tenant.
//
// One error covers "no such user", "a user of another tenant", "deleted" and "suspended".
// The caller is authenticated but not allowed in, and which of the four applies is not
// something to tell it.
var ErrNotMember = errors.New("authz: not an active member of this tenant")

// MemberLookup reads a membership. The interface sits here, on the consumer side.
type MemberLookup interface {
	GetMembership(ctx context.Context, tenantID, userID string) (*user.Membership, error)
}

// VerifyMember returns the role the subject holds in the tenant now.
//
// The role comes from the tenant's record, never from the token. The claim is what the
// subject was when it logged in; trusting it would let a demoted administrator keep
// administering until the token expired.
//
// Complexity: one primary-key lookup, O(log n).
func VerifyMember(ctx context.Context, members MemberLookup, tenantID, userID string) (user.UserRole, error) {
	// A malformed id cannot name a member. Checking it here keeps it from reaching the
	// database as a type error, which would surface as a 500 rather than a refusal.
	if uuid.Validate(tenantID) != nil || uuid.Validate(userID) != nil {
		return "", ErrNotMember
	}

	membership, err := members.GetMembership(ctx, tenantID, userID)

	switch {
	case errors.Is(err, user.ErrUserNotFound):
		return "", ErrNotMember
	case err != nil:
		return "", fmt.Errorf("authz: could not read membership: %w", err)
	case membership.Status != user.UserStatusActive:
		return "", ErrNotMember
	}

	return membership.Role, nil
}
