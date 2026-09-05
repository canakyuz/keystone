package middleware

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProbeApp, extractTenantID'nin sonucunu gövdede döndüren küçük bir uygulama
// kurar. Böylece tenant seçiminin kaynağını doğrudan sınayabiliriz.
func newProbeApp(authTenantID string) *fiber.App {
	app := fiber.New()

	app.Get("/probe", func(c *fiber.Ctx) error {
		// AuthMiddleware doğrulanmış JWT claim'ini böyle yazıyor.
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

// TestExtractTenantID_HeaderCannotOverrideJWT, kapatılan ayrıcalık yükseltme
// açığının geri gelmesini engeller.
//
// Açık şuydu: extractTenantID, c.Locals("user") anahtarını okuyup
// map[string]interface{}'e çevirmeye çalışıyordu. AuthMiddleware böyle bir
// anahtar hiç yazmıyor, claim'i doğrudan c.Locals("tenant_id") olarak koyuyor.
// Bu yüzden JWT yolu hiçbir zaman çalışmadı ve her istek sessizce X-Tenant-ID
// header'ına düştü. Geçerli token taşıyan herhangi bir kullanıcı, header'ı
// değiştirerek başka bir tenant'ın verisini okuyabiliyordu.
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

	t.Run("uretimde JWT yoksa header kabul edilmez", func(t *testing.T) {
		AllowUntrustedTenantSource(false)
		app := newProbeApp("")

		got := probe(t, app, "/probe", map[string]string{"X-Tenant-ID": attackTenant})

		assert.Empty(t, got, "kimlik dogrulanmadan header ile tenant secildi")
	})

	t.Run("development'ta JWT yoksa header kullanilabilir", func(t *testing.T) {
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
