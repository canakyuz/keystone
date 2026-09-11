package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/canakyuz/keystone/test/helpers"
)

// TestAuthMe_ReturnsTheCaller: /auth/me is the one call a client makes to learn who it is.
func TestAuthMe_ReturnsTheCaller(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-me").ID
	member := helpers.CreateTestUser(t, h.admin, tenantID, "me@authz-me.test", "viewer")

	status, body := h.send(t, http.MethodGet, "/api/v1/auth/me", tokenFor(t, tenantID, member.ID, "viewer"), "")

	assert.Equal(t, http.StatusOK, status, "the caller could not read itself: %s", body)
	assert.Contains(t, body, "me@authz-me.test")
}
