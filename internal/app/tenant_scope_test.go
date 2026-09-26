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
	"github.com/canakyuz/keystone/internal/middleware"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
)

// TestTenantScope_DatabaseFailureIsNotShownToTheClient verifies a failure to load the tenant
// is answered as a generic 500, not as a 403 carrying the driver's message.
//
// The repository is real and points at a port nothing listens on, so the error is the one
// production would see during an outage, host and port included.
func TestTenantScope_DatabaseFailureIsNotShownToTheClient(t *testing.T) {
	unreachable, err := sql.Open("postgres",
		"host=127.0.0.1 port=1 user=keystone dbname=keystone sslmode=disable connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = unreachable.Close() })

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
