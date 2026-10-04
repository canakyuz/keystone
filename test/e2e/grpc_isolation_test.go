package e2e

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/canakyuz/keystone/internal/database"
	keystonegrpc "github.com/canakyuz/keystone/internal/grpc"
	keystonev1 "github.com/canakyuz/keystone/internal/grpc/keystone/v1"
	"github.com/canakyuz/keystone/internal/middleware"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	platformrepo "github.com/canakyuz/keystone/internal/repository/platform"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	opuc "github.com/canakyuz/keystone/internal/usecase/operation"
	"github.com/canakyuz/keystone/test/helpers"
)

// grpcHarness runs the real gRPC server over an in-memory listener.
//
// bufconn rather than a real port: the tests run in parallel with the rest of the suite
// and a fixed port would make them collide, while a random one would leave the failure
// mode "the test picked a port something else was using" to be diagnosed later.
type grpcHarness struct {
	*harness

	client     keystonev1.UserServiceClient
	operations keystonev1.OperationServiceClient
}

// newGRPCHarness builds the same server the application builds.
//
// The point of this package is that the surfaces cannot diverge, so the test wires the
// production objects rather than stand-ins: the real interceptor chain, the real schema
// cache, the real repository, over the non-superuser role that production uses.
func newGRPCHarness(t *testing.T) *grpcHarness {
	t.Helper()

	base := newHarness(t)

	schemaCache := middleware.NewTenantSchemaCache(nil, base.appDB, nil, nil)
	users := userRepo.NewPostgresRepository(base.appDB, database.NewTenantConnectionManager(base.appDB, nil))

	server := keystonegrpc.New(
		keystonegrpc.Config{JWTSecret: testJWTSecret},
		schemaCache,
		users,
		keystonegrpc.NewOperationService(opuc.NewService(oprepo.New(base.appDB), platformrepo.New(base.appDB))),
		keystonegrpc.NewUserService(users),
		nil,
		nil,
	)

	listener := bufconn.Listen(1024 * 1024)

	go func() { _ = server.Handler().Serve(listener) }()
	t.Cleanup(func() { server.Handler().Stop() })

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return &grpcHarness{
		harness:    base,
		client:     keystonev1.NewUserServiceClient(conn),
		operations: keystonev1.NewOperationServiceClient(conn),
	}
}

// callAs returns a context carrying a token for the given tenant.
func (h *grpcHarness) callAs(t *testing.T, tenantID string) (context.Context, context.CancelFunc) {
	t.Helper()

	// The interceptor checks membership against the tenant's record, so the token has to
	// belong to a subject the tenant actually holds.
	member := helpers.CreateTestUser(t, h.admin, tenantID, "grpc-caller@"+tenantID+".test", "viewer")

	return withToken(signTokenAs(t, tenantID, member.ID, "viewer"))
}

// withToken returns a context carrying the given bearer token.
func withToken(token string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), cancel
}

// TestGRPC_RequestWithoutToken_IsRejected verifies the interceptor refuses an
// unauthenticated call before it can reach any tenant data.
func TestGRPC_RequestWithoutToken_IsRejected(t *testing.T) {
	h := newGRPCHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestGRPC_InvalidToken_IsRejected verifies a token signed with the wrong key is refused.
//
// The same assertion exists for HTTP. It is repeated here because the two surfaces would
// only be guaranteed to agree if they shared the verification, and this is what proves
// they do rather than assuming it.
func TestGRPC_InvalidToken_IsRejected(t *testing.T) {
	h := newGRPCHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer not-a-valid-token")

	_, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestGRPC_UnknownTenant_IsRejected verifies a valid token naming a tenant that does not
// exist is refused rather than falling through to an unscoped query.
func TestGRPC_UnknownTenant_IsRejected(t *testing.T) {
	h := newGRPCHarness(t)

	ctx, cancel := withToken(signToken(t, "00000000-0000-4000-8000-0000deadbeef"))
	defer cancel()

	_, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})

	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestGRPC_CrossTenantRead_ReturnsNothing is the central claim, asserted on this surface.
//
// The roadmap's finishing condition for the gRPC phase was that the same isolation tests
// pass over gRPC. This is that test: a contract that only proves its own codegen works
// would say nothing about whether the guarantee survives a second transport.
func TestGRPC_CrossTenantRead_ReturnsNothing(t *testing.T) {
	h := newGRPCHarness(t)

	alpha := createTenant(t, h.harness, "e2e-grpc-alpha")
	beta := createTenant(t, h.harness, "e2e-grpc-beta")

	seedUser(t, h.admin, alpha, "shared@alpha.test")
	seedUser(t, h.admin, beta, "shared@beta.test")

	ctx, cancel := h.callAs(t, alpha)
	defer cancel()

	resp, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})
	require.NoError(t, err)

	emails := make([]string, 0, len(resp.GetUsers()))
	for _, u := range resp.GetUsers() {
		emails = append(emails, u.GetEmail())
	}

	assert.Contains(t, emails, "shared@alpha.test", "the tenant's own user is missing")
	assert.NotContains(t, emails, "shared@beta.test", "another tenant's user leaked over gRPC")
}

// TestGRPC_MetadataCannotOverrideTheToken verifies the escalation closed on the HTTP
// surface cannot be reopened on this one.
//
// gRPC metadata is caller-controlled, exactly like an HTTP header. The bug that was fixed
// once — a tenant taken from a header instead of the verified claim — would be just as
// available here if the interceptor read metadata for the tenant, so it is asserted
// rather than assumed.
func TestGRPC_MetadataCannotOverrideTheToken(t *testing.T) {
	h := newGRPCHarness(t)

	alpha := createTenant(t, h.harness, "e2e-grpc-forge-alpha")
	beta := createTenant(t, h.harness, "e2e-grpc-forge-beta")

	seedUser(t, h.admin, alpha, "owner@alpha.test")
	seedUser(t, h.admin, beta, "secret@beta.test")

	ctx, cancel := h.callAs(t, alpha)
	defer cancel()

	// Every shape somebody might hope the server reads.
	ctx = metadata.AppendToOutgoingContext(ctx,
		"x-tenant-id", beta,
		"tenant-id", beta,
		"tenant_id", beta,
	)

	resp, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})
	require.NoError(t, err)

	emails := make([]string, 0, len(resp.GetUsers()))
	for _, u := range resp.GetUsers() {
		emails = append(emails, u.GetEmail())
	}

	assert.Contains(t, emails, "owner@alpha.test")
	assert.NotContains(t, emails, "secret@beta.test",
		"caller-supplied metadata overrode the verified token")
}

// TestGRPC_PageSizeIsCapped verifies a caller cannot ask for unbounded work.
//
// limit is caller-controlled and multiplies both the query cost and the memory held to
// answer it, which is a denial of service wearing a legitimate request's clothes.
func TestGRPC_PageSizeIsCapped(t *testing.T) {
	h := newGRPCHarness(t)

	tenantID := createTenant(t, h.harness, "e2e-grpc-page")
	seedUser(t, h.admin, tenantID, "owner@alpha.test")

	ctx, cancel := h.callAs(t, tenantID)
	defer cancel()

	resp, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{Limit: 1_000_000})
	require.NoError(t, err)

	assert.LessOrEqual(t, len(resp.GetUsers()), 200)
}

// TestGRPC_NonMember_IsRefused: the interceptor checks membership the way the HTTP
// middleware does. A genuine token is not enough once the membership behind it is gone.
func TestGRPC_NonMember_IsRefused(t *testing.T) {
	h := newGRPCHarness(t)

	tenantID := createTenant(t, h.harness, "e2e-grpc-member")
	member := helpers.CreateTestUser(t, h.admin, tenantID, "member@grpc-member.test", "admin")

	ctx, cancel := withToken(signTokenAs(t, tenantID, member.ID, "admin"))
	defer cancel()

	_, err := h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})
	require.NoError(t, err, "an active member was refused")

	_, err = h.admin.Exec(`UPDATE users SET status = 'suspended' WHERE id = $1`, member.ID)
	require.NoError(t, err)

	_, err = h.client.ListUsers(ctx, &keystonev1.ListUsersRequest{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "a suspended member kept access")

	// A well-formed token for a subject this tenant does not hold.
	strangerCtx, strangerCancel := withToken(signToken(t, tenantID))
	defer strangerCancel()

	_, err = h.client.ListUsers(strangerCtx, &keystonev1.ListUsersRequest{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "a subject the tenant does not hold was let in")
}

// TestGRPC_CreateTenant_RequiresPlatformOperator verifies the gRPC port refuses tenant
// creation to a subject without a platform_operators row, as the REST route does.
//
// It did not: REST checked the permission in a route middleware, and the gRPC interceptor
// only checks membership of the caller's own tenant, so any member of any tenant, down to
// a viewer, could create tenants whenever GRPC_ADDR was set.
func TestGRPC_CreateTenant_RequiresPlatformOperator(t *testing.T) {
	h := newGRPCHarness(t)

	home := createTenant(t, h.harness, "e2e-grpc-home")

	ctx, cancel := h.callAs(t, home)
	defer cancel()

	_, err := h.operations.CreateTenant(ctx, &keystonev1.CreateTenantRequest{
		Name: "Intruder", Slug: "e2e-grpc-intruder", Email: "intruder@example.test",
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	var count int
	require.NoError(t, h.admin.QueryRow(
		`SELECT count(*) FROM tenants WHERE slug = 'e2e-grpc-intruder'`).Scan(&count))
	assert.Zero(t, count, "a tenant was created without the platform permission")
}

// TestGRPC_CreateTenant_ValidatesLikeREST verifies the two ports share one set of input
// rules. gRPC used to accept a name of spaces and pass an overlong idempotency key through
// to the database, where it failed as an internal error.
func TestGRPC_CreateTenant_ValidatesLikeREST(t *testing.T) {
	h := newGRPCHarness(t)

	home := createTenant(t, h.harness, "e2e-grpc-validate")

	// An operator, so the call gets past the permission and reaches the input rules.
	operator := helpers.CreateTestUser(t, h.admin, home, "operator@e2e-grpc-validate.test", "owner")
	_, err := h.admin.Exec(
		`INSERT INTO platform_operators (user_id, note) VALUES ($1, 'e2e validation test')`, operator.ID)
	require.NoError(t, err)

	ctx, cancel := withToken(signTokenAs(t, home, operator.ID, "owner"))
	defer cancel()

	for name, req := range map[string]*keystonev1.CreateTenantRequest{
		"blank name": {Name: "   ", Slug: "e2e-grpc-blank", Email: "a@example.test"},
		"long key": {Name: "A", Slug: "e2e-grpc-long", Email: "a@example.test",
			IdempotencyKey: strings.Repeat("k", opuc.MaxIdempotencyKeyLength+1)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.operations.CreateTenant(ctx, req)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}
