package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/test/helpers"
)

// TestRegister_IsNotAWayIn: nobody creates an account in a tenant by knowing its id.
//
// The endpoint that did this was removed rather than repaired. It accepted any tenant id
// from an anonymous caller, and it only ever failed because it ran without the tenant
// context; fixing that alone would have opened the door it was keeping shut by accident.
func TestRegister_IsNotAWayIn(t *testing.T) {
	h := newAuthzHarness(t)
	tenantID := helpers.CreateTestTenant(t, h.admin, "register-closed").ID

	body := `{"tenant_id":"` + tenantID + `","email":"stranger@register.test","password":"password123","first_name":"A","last_name":"B"}`
	status, resp := h.send(t, http.MethodPost, "/api/v1/auth/register", "", body)

	// 401 or 404, and either is the right answer. With the route gone the path falls under the
	// authenticated /auth group, whose middleware answers first; if the routes are ever
	// reordered it becomes a plain 404. What must not happen is a handler running.
	assert.Contains(t, []int{http.StatusUnauthorized, http.StatusNotFound}, status,
		"anonymous registration still answers: %s", resp)

	var count int
	require.NoError(t, h.admin.QueryRow(`SELECT count(*) FROM users WHERE email = 'stranger@register.test'`).Scan(&count))
	assert.Zero(t, count, "an anonymous caller created an account")
}
