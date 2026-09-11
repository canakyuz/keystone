package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
)

// TestOperations_WorkUnderTheApplicationRole drives provisioning, the status lookup and an
// idempotent retry over the role production connects with.
//
// The repository's own tests use the superuser connection, which RLS never applies to.
// Over that connection all three worked. Over this one, the first insert was refused by
// the tenant policy, so POST /tenants failed on every request, and the lookup matched no
// row, so every poll answered 404.
func TestOperations_WorkUnderTheApplicationRole(t *testing.T) {
	h := newHarness(t)
	repo := oprepo.New(h.appDB)
	ctx := context.Background()

	const (
		creator  = "00000000-0000-4000-8000-0000000000a1"
		stranger = "00000000-0000-4000-8000-0000000000a2"
	)

	req := oprepo.ProvisionRequest{
		Name:           "Role Probe",
		Slug:           "role-probe",
		Email:          "probe@example.com",
		CreatedBy:      creator,
		Scope:          "subject:" + creator,
		IdempotencyKey: "role-probe-1",
		RequestBody:    []byte(`{"slug":"role-probe"}`),
	}

	first, err := repo.CreateTenantProvision(ctx, req)
	require.NoError(t, err, "provisioning could not be accepted under the application role")

	op, err := repo.GetOperation(ctx, first.Operation.ID, creator)
	require.NoError(t, err, "the creator could not read its own operation")
	assert.Equal(t, first.TenantID, op.TenantID)

	// Visibility without tenant context is the creator's alone. An operation id travels in
	// a Location header and in logs; knowing it must not be enough to read it.
	_, err = repo.GetOperation(ctx, first.Operation.ID, stranger)
	assert.ErrorIs(t, err, domain.ErrNotFound, "another subject read the operation")

	_, err = repo.GetOperation(ctx, first.Operation.ID, "")
	assert.ErrorIs(t, err, domain.ErrNotFound, "a lookup with no subject read the operation")

	replay, err := repo.CreateTenantProvision(ctx, req)
	require.NoError(t, err, "an idempotent retry failed under the application role")
	assert.True(t, replay.Replayed, "the retry was not recognised as a replay")
	assert.Equal(t, first.Operation.ID, replay.Operation.ID)
}
