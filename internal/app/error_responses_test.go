package app

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/database"
	tenantHandler "github.com/canakyuz/keystone/internal/handler/tenant"
	"github.com/canakyuz/keystone/internal/middleware"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	"github.com/canakyuz/keystone/pkg/validator"
	"github.com/canakyuz/keystone/test/helpers"
)

// unreachableDB is a pool pointing at a port nothing listens on. The errors it produces
// are the ones an outage produces, host and port included.
func unreachableDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres",
		"host=127.0.0.1 port=1 user=keystone dbname=keystone sslmode=disable connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// currentTenant serves GET /current through the real handler, usecase and repository.
func currentTenant(t *testing.T, db *sql.DB, tenantID string) (int, string) {
	t.Helper()

	service := tenantUsecase.NewService(tenantRepo.NewPostgresRepository(db), validator.New(), nil, nil)
	h := tenantHandler.NewHandler(service)

	app := fiber.New(fiber.Config{ErrorHandler: customErrorHandler})
	app.Get("/current", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenantID)
		return h.GetCurrent(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/current", nil), 5000)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(body)
}

// TestHandlers_DatabaseFailureIsNotShownToTheClient verifies a handler answers an
// infrastructure failure with a generic 500. It used to write err.Error() under the status
// meant for the expected failure, so an outage came back as a 404 naming the database host.
func TestHandlers_DatabaseFailureIsNotShownToTheClient(t *testing.T) {
	status, body := currentTenant(t, unreachableDB(t), "0f2f7b52-0000-4000-8000-000000000001")

	assert.Equal(t, http.StatusInternalServerError, status, "an outage is not a missing tenant")
	assert.NotContains(t, body, "127.0.0.1")
}

// TestHandlers_DomainErrorsStillReachTheClient verifies the expected failure keeps its status
// and its message: hiding infrastructure detail must not hide what the caller can act on.
func TestHandlers_DomainErrorsStillReachTheClient(t *testing.T) {
	admin := helpers.SetupTestDB(t)
	appDB := helpers.SetupAppRoleDB(t, admin)

	status, body := currentTenant(t, appDB, "0f2f7b52-0000-4000-8000-00000000dead")

	assert.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, body, "tenant not found")
}

// TestTenantScope_DatabaseFailureIsNotShownToTheClient verifies a failure to load the tenant
// is answered as a generic 500, not as a 403 carrying the driver's message.
//
// The repository is real and points at a port nothing listens on, so the error is the one
// production would see during an outage, host and port included.
func TestTenantScope_DatabaseFailureIsNotShownToTheClient(t *testing.T) {
	unreachable := unreachableDB(t)

	app := fiber.New(fiber.Config{ErrorHandler: customErrorHandler})
	app.Get("/scoped",
		func(c *fiber.Ctx) error {
			c.Locals("tenant_id", "0f2f7b52-0000-4000-8000-000000000001")
			return c.Next()
		},
		middleware.TenantScope(tenantRepo.NewPostgresRepository(unreachable), database.NewTenantManager(unreachable)),
		func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) },
	)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/scoped", nil), 5000)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "an outage is not a permission refusal")
	assert.NotContains(t, string(body), "127.0.0.1")
	assert.NotContains(t, string(body), "failed to get tenant")
}
