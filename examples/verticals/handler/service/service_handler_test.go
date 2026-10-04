package service

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

	serviceRepo "github.com/canakyuz/keystone/examples/verticals/repository/service"
	serviceUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/service"
	"github.com/canakyuz/keystone/pkg/logger"
)

// TestGetByID_DatabaseFailureIsNotShownToTheClient runs the real handler, usecase and
// repository against a port nothing listens on, on a Fiber app with the default error
// handler. The handler used to write err.Error(), which put the driver's message, host and
// port included, in the response; and the default handler would do the same with any error
// returned to it, so the handler has to answer on its own.
func TestGetByID_DatabaseFailureIsNotShownToTheClient(t *testing.T) {
	db, err := sql.Open("postgres",
		"host=127.0.0.1 port=1 user=keystone dbname=keystone sslmode=disable connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	h := NewServiceHandler(serviceUsecase.NewServiceService(serviceRepo.NewServicePostgresRepository(db), *logger.Default()))

	app := fiber.New()
	app.Get("/services/:id", h.GetByID)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/services/0f2f7b52-0000-4000-8000-000000000001", nil), 5000)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.NotContains(t, string(body), "127.0.0.1")
}
