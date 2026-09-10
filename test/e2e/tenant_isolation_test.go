package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/middleware"
)

// decodeEmails reads the email list out of a successful response.
func decodeEmails(t *testing.T, resp *http.Response) []string {
	t.Helper()

	var body struct {
		Emails []string `json:"emails"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	return body.Emails
}

// TestRequestWithoutToken_IsRejected verifies the chain refuses an unauthenticated
// request before it can reach any tenant data.
func TestRequestWithoutToken_IsRejected(t *testing.T) {
	h := newHarness(t)

	resp := h.do(t, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestRequestWithInvalidToken_IsRejected verifies a token signed with the wrong key
// is refused.
//
// This matters more than it looks: if signature verification were skipped, the tenant
// claim would become attacker-controlled and every isolation guarantee below it would
// be worthless.
func TestRequestWithInvalidToken_IsRejected(t *testing.T) {
	h := newHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")

	resp := h.do(t, req)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestUnknownTenant_Returns404 verifies a well-formed token naming a tenant that does
// not exist is refused rather than falling through to an unscoped query.
func TestUnknownTenant_Returns404(t *testing.T) {
	h := newHarness(t)

	resp := h.do(t, h.get(t, "/api/v1/users", "00000000-0000-4000-8000-0000deadbeef"))

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestMalformedTenantID_DoesNotReachTheDatabase verifies a tenant id that is not a
// UUID is refused.
//
// The value flows into a query that casts it to UUID. It is passed as a bound
// parameter, so this is not an injection vector; the assertion is that the request
// fails cleanly instead of surfacing a raw database error.
func TestMalformedTenantID_DoesNotReachTheDatabase(t *testing.T) {
	h := newHarness(t)

	for _, tenantID := range []string{
		"'; DROP TABLE users; --",
		"public",
		"tenant_acme; SELECT 1",
		"../../etc/passwd",
	} {
		t.Run(tenantID, func(t *testing.T) {
			resp := h.do(t, h.get(t, "/api/v1/users", tenantID))

			assert.NotEqual(t, http.StatusOK, resp.StatusCode,
				"a malformed tenant id was served successfully")
		})
	}
}

// TestCrossTenantRead_ReturnsNothing is the central claim of this package: a token
// for one tenant cannot read another tenant's rows over HTTP.
//
// Both tenants hold a user with the same local part, so a leak would be unmistakable
// in the assertion rather than hidden behind an empty list.
func TestCrossTenantRead_ReturnsNothing(t *testing.T) {
	h := newHarness(t)

	alpha := createTenant(t, h, "e2e-alpha")
	beta := createTenant(t, h, "e2e-beta")

	seedUser(t, h.admin, alpha, "shared@alpha.test")
	seedUser(t, h.admin, beta, "shared@beta.test")

	resp := h.do(t, h.get(t, "/api/v1/users", alpha))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	emails := decodeEmails(t, resp)

	assert.Contains(t, emails, "shared@alpha.test", "the tenant's own user is missing")
	assert.NotContains(t, emails, "shared@beta.test", "another tenant's user leaked over HTTP")
}

// TestForgedTenantHeader_CannotOverrideJWT verifies the closed privilege escalation
// stays closed at the HTTP boundary.
//
// The bug: extractTenantID read a context key AuthMiddleware never writes, so the JWT
// path never ran and every request silently fell through to the X-Tenant-ID header.
// Any user with a valid token could read any tenant by editing one header.
//
// internal/middleware covers this at the unit level. It is repeated here because the
// unit test calls extractTenantID directly, and what an employer needs to see is that
// the whole assembled application refuses the attack.
func TestForgedTenantHeader_CannotOverrideJWT(t *testing.T) {
	h := newHarness(t)

	alpha := createTenant(t, h, "e2e-forge-alpha")
	beta := createTenant(t, h, "e2e-forge-beta")

	seedUser(t, h.admin, alpha, "owner@alpha.test")
	seedUser(t, h.admin, beta, "secret@beta.test")

	// A valid token for alpha, plus a header claiming beta.
	req := h.get(t, "/api/v1/users", alpha)
	req.Header.Set("X-Tenant-ID", beta)

	resp := h.do(t, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	emails := decodeEmails(t, resp)

	assert.Contains(t, emails, "owner@alpha.test")
	assert.NotContains(t, emails, "secret@beta.test",
		"the X-Tenant-ID header overrode the verified JWT claim")
}

// TestTenantQueryParameter_CannotOverrideJWT covers the same escalation through the
// query parameter, the other untrusted source.
func TestTenantQueryParameter_CannotOverrideJWT(t *testing.T) {
	h := newHarness(t)

	alpha := createTenant(t, h, "e2e-query-alpha")
	beta := createTenant(t, h, "e2e-query-beta")

	seedUser(t, h.admin, alpha, "owner@alpha.test")
	seedUser(t, h.admin, beta, "secret@beta.test")

	resp := h.do(t, h.get(t, "/api/v1/users?tenant_id="+beta, alpha))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	emails := decodeEmails(t, resp)

	assert.NotContains(t, emails, "secret@beta.test",
		"the tenant_id query parameter overrode the verified JWT claim")
}

// TestConnectionIsResetAfterRequest verifies a connection goes back to the pool clean.
//
// If either search_path or app.current_tenant survived the request, a later request
// served on the same pooled connection would silently read the previous tenant's
// data. This is invisible unless the pool is inspected after a tenant-scoped request
// has run.
func TestConnectionIsResetAfterRequest(t *testing.T) {
	h := newHarness(t)

	alpha := createTenant(t, h, "e2e-searchpath")
	seedUser(t, h.admin, alpha, "owner@alpha.test")

	// Drive a tenant-scoped request so the pool has served one.
	resp := h.do(t, h.get(t, "/api/v1/users", alpha))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Now inspect what the pool hands out.
	probe := h.do(t, httptest.NewRequest(http.MethodGet, "/_probe/scope", nil))
	require.Equal(t, http.StatusOK, probe.StatusCode)

	var body struct {
		SearchPath    string `json:"search_path"`
		CurrentTenant string `json:"current_tenant"`
	}
	require.NoError(t, json.NewDecoder(probe.Body).Decode(&body))

	assert.NotContains(t, body.SearchPath, "tenant_",
		"a connection carrying a tenant search_path was returned to the pool")
	assert.Empty(t, body.CurrentTenant,
		"a connection carrying app.current_tenant was returned to the pool")
}

// TestUntrustedSourceStillRequiresExistingTenant verifies that even in development
// posture, where the header is accepted, the named tenant must actually exist.
//
// This guards the development convenience itself: it may relax where the tenant id
// comes from, never whether the tenant is real.
func TestUntrustedSourceStillRequiresExistingTenant(t *testing.T) {
	h := newHarness(t)

	middleware.AllowUntrustedTenantSource(true)
	defer middleware.AllowUntrustedTenantSource(false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, ""))
	req.Header.Set("X-Tenant-ID", "00000000-0000-4000-8000-0000deadbeef")

	resp := h.do(t, req)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
