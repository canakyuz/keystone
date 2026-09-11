package middleware

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProbeApp builds a tiny application that returns whatever extractTenantID
// produced in the response body, so the source of the tenant choice can be asserted
// directly.
func newProbeApp(authTenantID string) *fiber.App {
	app := fiber.New()

	app.Get("/probe", func(c *fiber.Ctx) error {
		// This is how AuthMiddleware writes the verified JWT claim.
		if authTenantID != "" {
			c.Locals("tenant_id", authTenantID)
		}
		return c.SendString(extractTenantID(c))
	})

	return app
}

func probe(t *testing.T, app *fiber.App, url string, headers map[string]string) string {
	t.Helper()

	req := httpGet(t, url)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return readAll(t, resp.Body)
}

// TestExtractTenantID_HeaderCannotOverrideJWT keeps the privilege escalation bug
// that was closed here from coming back.
//
// The bug: extractTenantID read the c.Locals("user") key and tried to assert it to
// map[string]interface{}. AuthMiddleware never writes such a key; it puts the claim
// directly into c.Locals("tenant_id"). So the JWT path never ran and every request
// silently fell through to the X-Tenant-ID header. Any user holding a valid token
// could read another tenant's data just by changing that header.
func TestExtractTenantID_HeaderCannotOverrideJWT(t *testing.T) {
	const (
		jwtTenant    = "11111111-1111-1111-1111-111111111111"
		attackTenant = "22222222-2222-2222-2222-222222222222"
	)

	t.Run("uretimde header JWT'yi ezemez", func(t *testing.T) {
		AllowUntrustedTenantSource(false)
		app := newProbeApp(jwtTenant)

		got := probe(t, app, "/probe", map[string]string{"X-Tenant-ID": attackTenant})

		assert.Equal(t, jwtTenant, got, "header JWT claim'ini ezdi")
	})

	t.Run("uretimde query JWT'yi ezemez", func(t *testing.T) {
		AllowUntrustedTenantSource(false)
		app := newProbeApp(jwtTenant)

		got := probe(t, app, "/probe?tenant_id="+attackTenant, nil)

		assert.Equal(t, jwtTenant, got, "query parametresi JWT claim'ini ezdi")
	})

	t.Run("in production the header is refused without a JWT", func(t *testing.T) {
		AllowUntrustedTenantSource(false)
		app := newProbeApp("")

		got := probe(t, app, "/probe", map[string]string{"X-Tenant-ID": attackTenant})

		assert.Empty(t, got, "kimlik dogrulanmadan header ile tenant secildi")
	})

	t.Run("in development the header is used without a JWT", func(t *testing.T) {
		AllowUntrustedTenantSource(true)
		t.Cleanup(func() { AllowUntrustedTenantSource(false) })

		app := newProbeApp("")

		got := probe(t, app, "/probe", map[string]string{"X-Tenant-ID": attackTenant})

		assert.Equal(t, attackTenant, got, "development kolayligi bozuldu")
	})

	t.Run("development'ta bile JWT header'dan onceliklidir", func(t *testing.T) {
		AllowUntrustedTenantSource(true)
		t.Cleanup(func() { AllowUntrustedTenantSource(false) })

		app := newProbeApp(jwtTenant)

		got := probe(t, app, "/probe", map[string]string{"X-Tenant-ID": attackTenant})

		assert.Equal(t, jwtTenant, got, "development'ta header JWT'yi ezdi")
	})
}
