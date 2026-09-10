// Package e2e exercises tenant isolation across the whole request path.
//
// The other isolation tests each cover one layer. test/security drives SQL directly
// to check the Row Level Security configuration; the repository tests check the
// queries. Neither covers the step in between: the middleware that turns an incoming
// request into a tenant schema. All isolation rests on that step, and until this
// package existed it had no test.
//
// So these tests start at the HTTP boundary and end at the database. They use the
// real JWT middleware, the real tenant context middleware, the real schema cache and
// the real repository, against real PostgreSQL.
package e2e

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/domain/user"
	"github.com/canakyuz/keystone/internal/middleware"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	"github.com/canakyuz/keystone/pkg/database"
	"github.com/canakyuz/keystone/test/helpers"
)

const testJWTSecret = "e2e-test-secret"

// harness holds everything a test needs to drive the application.
type harness struct {
	app   *fiber.App
	admin *sql.DB
	appDB *sql.DB
}

// newHarness builds the application under test.
//
// Two database handles are kept on purpose. admin is the superuser connection used to
// insert fixtures; appDB is the non-superuser, table-owning role the application
// itself connects with. RLS is never enforced on a superuser connection, so serving
// requests over admin would make every isolation assertion here meaningless.
func newHarness(t *testing.T) *harness {
	t.Helper()

	admin := helpers.SetupTestDB(t)
	appDB := helpers.SetupAppRoleDB(t, admin)

	// Production posture: the JWT claim is the only accepted tenant source.
	middleware.AllowUntrustedTenantSource(false)
	t.Cleanup(func() { middleware.AllowUntrustedTenantSource(false) })

	schemaCache := middleware.NewTenantSchemaCache(nil, appDB, nil, nil)
	users := userRepo.NewPostgresRepository(appDB, database.NewTenantConnectionManager(appDB, nil))

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	protected := app.Group("/api/v1",
		middleware.AuthMiddleware(testJWTSecret),
		middleware.TenantContextMiddleware(schemaCache),
	)

	// The handler deliberately passes c.UserContext(), the context the tenant
	// middleware wrote the schema into. Passing anything else is the mistake this
	// package exists to catch.
	protected.Get("/users", func(c *fiber.Ctx) error {
		list, _, err := users.List(c.UserContext(), middleware.GetTenantID(c), userRepo.ListFilters{Limit: 100})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		emails := make([]string, 0, len(list))
		for _, u := range list {
			emails = append(emails, u.Email)
		}

		return c.JSON(fiber.Map{"emails": emails})
	})

	// Reports the tenant-scoping state of a connection taken from the pool. Used to
	// prove the pool is not handed back a connection still scoped to a tenant.
	//
	// Both values matter and for different reasons: search_path decides which schema
	// unqualified names resolve to, app.current_tenant decides what the RLS policies
	// let through. A connection leaking either one lets the next request read the
	// previous tenant's data.
	app.Get("/_probe/scope", func(c *fiber.Ctx) error {
		conn, err := appDB.Conn(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer conn.Close()

		var searchPath, currentTenant string
		if err := conn.QueryRowContext(c.Context(),
			"SELECT current_setting('search_path'), current_setting('app.current_tenant', TRUE)",
		).Scan(&searchPath, &currentTenant); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"search_path": searchPath, "current_tenant": currentTenant})
	})

	return &harness{app: app, admin: admin, appDB: appDB}
}

// do runs one request through the application.
func (h *harness) do(t *testing.T, req *http.Request) *http.Response {
	t.Helper()

	resp, err := h.app.Test(req, 10_000)
	require.NoError(t, err)

	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

// get builds an authenticated GET request for the given tenant.
func (h *harness) get(t *testing.T, path, tenantID string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, tenantID))

	return req
}

// signToken issues a valid JWT carrying the given tenant claim.
func signToken(t *testing.T, tenantID string) string {
	t.Helper()

	claims := middleware.JWTClaims{
		UserID:   "00000000-0000-4000-8000-00000000e2e0",
		TenantID: tenantID,
		Email:    "e2e@example.com",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	return signed
}

// seedUser inserts a user row for the given tenant using the admin connection.
func seedUser(t *testing.T, admin *sql.DB, tenantID, email string) {
	t.Helper()

	_, err := admin.Exec(`
		INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, role, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, 'x', 'E2E', 'User', $3, $4, NOW(), NOW())
	`, tenantID, email, user.RoleAdmin, user.UserStatusActive)
	require.NoError(t, err)
}

// createTenant creates a tenant with its schema and returns its id.
func createTenant(t *testing.T, h *harness, slug string) string {
	t.Helper()

	return helpers.CreateTestTenant(t, h.admin, slug).ID
}
