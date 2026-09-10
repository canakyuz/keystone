package e2e

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userHandler "github.com/canakyuz/keystone/internal/handler/user"
	"github.com/canakyuz/keystone/internal/middleware"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	userUsecase "github.com/canakyuz/keystone/internal/usecase/user"
	"github.com/canakyuz/keystone/pkg/database"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/validator"
)

// TestRealUserHandler_ResolvesTenantSchema drives the production handler, not a stand-in.
//
// WHY this exists separately from the rest of this package: the other tests wire their
// own route onto the middleware, which proves the middleware works but says nothing
// about whether the real handlers use it correctly. They did not. Every endpoint under
// /api/v1/users passed c.Context() into the service, while the tenant middleware writes
// the schema into c.UserContext(), so the repository never saw a schema and the whole
// resource answered 500.
//
// A test that builds its own handler would have stayed green through all of it. This one
// builds the same objects internal/app/app.go builds, in the same order, so the wiring
// itself is under test.
func TestRealUserHandler_ResolvesTenantSchema(t *testing.T) {
	h := newHarness(t)

	tenantID := createTenant(t, h, "e2e-real-wiring")
	seedUser(t, h.admin, tenantID, "owner@real.test")

	// The same construction as the composition root.
	repository := userRepo.NewPostgresRepository(h.appDB, database.NewTenantConnectionManager(h.appDB, nil))
	service := userUsecase.NewService(repository, validator.New(), logger.Default(), testJWTSecret)
	handler := userHandler.NewHandler(service)

	schemaCache := middleware.NewTenantSchemaCache(nil, h.appDB, nil, nil)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Group("/api/v1/users",
		middleware.AuthMiddleware(testJWTSecret),
		middleware.TenantContextMiddleware(schemaCache),
	).Get("/", handler.List)

	req := h.get(t, "/api/v1/users", tenantID)

	resp, err := app.Test(req, 10_000)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode,
		"the real handler failed to resolve the tenant schema: %s", body)

	assert.Contains(t, string(body), "owner@real.test",
		"the tenant's own user is missing from the response")
}
