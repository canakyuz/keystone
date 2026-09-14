package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	webhookRepo "github.com/canakyuz/keystone/internal/repository/webhook"
	"github.com/canakyuz/keystone/test/helpers"
)

// TestWebhookEndpoints_AreManagedByTheirTenantsAdministrators covers who may register a
// destination, which destinations are refused, what happens to the secret, and that another
// tenant can neither see nor change one.
func TestWebhookEndpoints_AreManagedByTheirTenantsAdministrators(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "hooks-own").ID
	other := helpers.CreateTestTenant(t, h.admin, "hooks-other").ID
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@hooks-own.test", "admin")
	viewer := helpers.CreateTestUser(t, h.admin, tenantID, "viewer@hooks-own.test", "viewer")
	outsider := helpers.CreateTestUser(t, h.admin, other, "admin@hooks-other.test", "admin")

	adminToken := tokenFor(t, tenantID, admin.ID, "admin")
	const path = "/api/v1/webhook-endpoints"

	status, body := h.send(t, http.MethodPost, path, tokenFor(t, tenantID, viewer.ID, "viewer"),
		`{"url":"https://hooks.example.com/keystone"}`)
	assert.Equal(t, http.StatusForbidden, status, "a viewer registered an endpoint: %s", body)

	for _, refused := range []string{
		"http://hooks.example.com/keystone",
		"https://169.254.169.254/latest/meta-data",
		"https://10.0.0.5/hook",
		"https://user:pass@hooks.example.com/keystone",
	} {
		status, body := h.send(t, http.MethodPost, path, adminToken, `{"url":"`+refused+`"}`)
		assert.Equal(t, http.StatusBadRequest, status, "%s was accepted: %s", refused, body)
	}

	status, body = h.send(t, http.MethodPost, path, adminToken,
		`{"url":"https://hooks.example.com/services/T000/secret-in-the-path"}`)
	require.Equal(t, http.StatusCreated, status, "a valid endpoint was refused: %s", body)

	var created struct {
		Data struct {
			ID         string `json:"id"`
			Secret     string `json:"secret"`
			SecretHint string `json:"secret_hint"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	require.Len(t, created.Data.Secret, 64, "the secret is not 32 random bytes")
	assert.Equal(t, created.Data.Secret[60:], created.Data.SecretHint)

	status, body = h.send(t, http.MethodGet, path, adminToken, "")
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, created.Data.ID)
	assert.NotContains(t, body, created.Data.Secret, "the secret was listed again")

	outsiderToken := tokenFor(t, other, outsider.ID, "admin")

	status, body = h.send(t, http.MethodGet, path, outsiderToken, "")
	require.Equal(t, http.StatusOK, status, body)
	assert.NotContains(t, body, created.Data.ID, "another tenant listed this endpoint")

	status, body = h.send(t, http.MethodPatch, path+"/"+created.Data.ID, outsiderToken, `{"active":false}`)
	assert.Equal(t, http.StatusNotFound, status, "another tenant changed this endpoint: %s", body)

	status, body = h.send(t, http.MethodPatch, path+"/"+created.Data.ID, adminToken, `{"active":false}`)
	require.Equal(t, http.StatusOK, status, body)

	var active bool
	require.NoError(t, h.admin.QueryRow(
		`SELECT active FROM tenant_webhook_endpoints WHERE id = $1`, created.Data.ID).Scan(&active))
	assert.False(t, active)

	rows, err := h.admin.Query(
		`SELECT action, metadata::text FROM audit_log WHERE tenant_id = $1 AND subject_id = $2`,
		tenantID, created.Data.ID)
	require.NoError(t, err)
	defer rows.Close()

	var actions []string
	var trail strings.Builder
	for rows.Next() {
		var action, metadata string
		require.NoError(t, rows.Scan(&action, &metadata))
		actions = append(actions, action)
		trail.WriteString(metadata)
	}
	require.NoError(t, rows.Err())

	assert.ElementsMatch(t, []string{"webhook.created", "webhook.disabled"}, actions)
	assert.Contains(t, trail.String(), "hooks.example.com")
	assert.NotContains(t, trail.String(), "secret-in-the-path", "the url's path reached the trail")
	assert.NotContains(t, trail.String(), created.Data.Secret, "the signing secret reached the trail")
}

// TestWebhookEndpoints_AreCapped: a tenant cannot register endpoints without limit, because
// each one multiplies the requests every event sends.
func TestWebhookEndpoints_AreCapped(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "hooks-cap").ID
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@hooks-cap.test", "admin")
	token := tokenFor(t, tenantID, admin.ID, "admin")

	for i := 0; i < webhookRepo.MaxPerTenant; i++ {
		status, body := h.send(t, http.MethodPost, "/api/v1/webhook-endpoints", token,
			fmt.Sprintf(`{"url":"https://hooks.example.com/%d"}`, i))
		require.Equal(t, http.StatusCreated, status, body)
	}

	status, body := h.send(t, http.MethodPost, "/api/v1/webhook-endpoints", token, `{"url":"https://hooks.example.com/over"}`)
	assert.Equal(t, http.StatusConflict, status, "the cap did not hold: %s", body)
}
