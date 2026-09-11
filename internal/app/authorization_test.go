package app

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/config"
	authHandler "github.com/canakyuz/keystone/internal/handler/auth"
	operationHandler "github.com/canakyuz/keystone/internal/handler/operation"
	tenantHandler "github.com/canakyuz/keystone/internal/handler/tenant"
	userHandler "github.com/canakyuz/keystone/internal/handler/user"
	"github.com/canakyuz/keystone/internal/middleware"
	operationRepo "github.com/canakyuz/keystone/internal/repository/operation"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	userUsecase "github.com/canakyuz/keystone/internal/usecase/user"
	"github.com/canakyuz/keystone/pkg/database"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/validator"
	"github.com/canakyuz/keystone/test/helpers"
)

const authzSecret = "authorization-test-secret"

type authzHarness struct {
	app   *fiber.App
	admin *sql.DB
}

// newAuthzHarness mounts the production route table over the non-superuser role.
//
// It calls setupRoutes and registerOperationRoutes rather than rebuilding the chain. The
// e2e package learned why: a test that wires its own routes proves the middleware works
// and says nothing about whether the application applies it. The handlers these tests do
// not reach are passed as nil; Fiber stores their method values without calling them.
func newAuthzHarness(t *testing.T) *authzHarness {
	t.Helper()

	admin := helpers.SetupTestDB(t)
	appDB := helpers.SetupAppRoleDB(t, admin)

	middleware.AllowUntrustedTenantSource(false)

	cfg := &config.Config{}
	cfg.Auth.JWTSecret = authzSecret

	log := logger.Default()
	tenants := tenantRepo.NewPostgresRepository(appDB)
	users := userRepo.NewPostgresRepository(appDB, database.NewTenantConnectionManager(appDB, nil))
	userService := userUsecase.NewService(users, validator.New(), log, authzSecret)
	membership := middleware.Membership(users)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	planRateLimit := func(c *fiber.Ctx) error { return c.Next() }

	registerOperationRoutes(app, authzSecret, membership,
		operationHandler.New(operationRepo.New(appDB), log))

	setupRoutes(app, cfg,
		authHandler.NewHandler(userService),
		tenantHandler.NewHandler(tenantUsecase.NewService(tenants, validator.New(), log, nil)),
		userHandler.NewHandler(userService),
		nil, nil, nil, nil,
		middleware.TenantContextMiddleware(middleware.NewTenantSchemaCache(nil, appDB, nil, nil)),
		middleware.TenantScope(tenants, database.NewTenantManager(appDB)),
		planRateLimit,
		membership,
	)

	return &authzHarness{app: app, admin: admin}
}

func (h *authzHarness) send(t *testing.T, method, path, token, body string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.app.Test(req, 10_000)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, string(raw)
}

func (h *authzHarness) roleOf(t *testing.T, userID string) string {
	t.Helper()

	var role string
	require.NoError(t, h.admin.QueryRow(`SELECT role FROM users WHERE id = $1`, userID).Scan(&role))

	return role
}

// tokenFor issues a genuine token. What it claims is up to the test; whether the claim is
// honoured is what is under test.
func tokenFor(t *testing.T, tenantID, userID, role string) string {
	t.Helper()

	claims := middleware.JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    userID + "@example.com",
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(authzSecret))
	require.NoError(t, err)

	return signed
}

type probe struct{ method, path, body string }

// TestAuthorization_TenantPathIsTheCallersOwn: the tenant handlers act on the id in the
// path. Before SameTenant, the owner of one tenant could read, rename and delete another.
func TestAuthorization_TenantPathIsTheCallersOwn(t *testing.T) {
	h := newAuthzHarness(t)

	own := helpers.CreateTestTenant(t, h.admin, "authz-own").ID
	other := helpers.CreateTestTenant(t, h.admin, "authz-other").ID
	owner := helpers.CreateTestUser(t, h.admin, own, "owner@authz-own.test", "owner")
	token := tokenFor(t, own, owner.ID, "owner")

	status, body := h.send(t, http.MethodGet, "/api/v1/tenants/"+own, token, "")
	require.Equal(t, http.StatusOK, status, "the caller's own tenant was refused: %s", body)

	for _, p := range []probe{
		{http.MethodGet, "/api/v1/tenants/" + other, ""},
		{http.MethodPatch, "/api/v1/tenants/" + other, `{"name":"Taken Over"}`},
		{http.MethodPatch, "/api/v1/tenants/" + other + "/branding", `{}`},
		{http.MethodDelete, "/api/v1/tenants/" + other, ""},
	} {
		status, body := h.send(t, p.method, p.path, token, p.body)
		assert.Equal(t, http.StatusNotFound, status, "%s %s: %s", p.method, p.path, body)
	}

	var name, tenantStatus string
	var deletedAt sql.NullTime
	require.NoError(t, h.admin.QueryRow(
		`SELECT name, status, deleted_at FROM tenants WHERE id = $1`, other,
	).Scan(&name, &tenantStatus, &deletedAt))

	assert.Equal(t, "Test Tenant authz-other", name, "another tenant was renamed")
	assert.Equal(t, "active", tenantStatus, "another tenant's status changed")
	assert.False(t, deletedAt.Valid, "another tenant was deleted")
}

// TestAuthorization_CrossTenantRoutesAreClosed: listing, counting and lifecycle changes
// act across tenants. Until a platform permission exists they are refused to everyone,
// including an owner acting on its own tenant.
func TestAuthorization_CrossTenantRoutesAreClosed(t *testing.T) {
	h := newAuthzHarness(t)

	own := helpers.CreateTestTenant(t, h.admin, "authz-platform").ID
	owner := helpers.CreateTestUser(t, h.admin, own, "owner@authz-platform.test", "owner")
	token := tokenFor(t, own, owner.ID, "owner")

	for _, p := range []probe{
		{http.MethodGet, "/api/v1/tenants", ""},
		{http.MethodGet, "/api/v1/tenants/stats", ""},
		{http.MethodGet, "/api/v1/tenants/slug/authz-platform", ""},
		{http.MethodPost, "/api/v1/tenants/" + own + "/suspend", `{"reason":"x"}`},
		{http.MethodPost, "/api/v1/tenants/" + own + "/activate", ""},
		{http.MethodPost, "/api/v1/tenants/" + own + "/upgrade", `{"plan":"enterprise"}`},
	} {
		status, body := h.send(t, p.method, p.path, token, p.body)
		assert.Equal(t, http.StatusForbidden, status, "%s %s: %s", p.method, p.path, body)
	}
}

// TestAuthorization_RoleComesFromTheTenantsRecord: the role a guard checks is the one the
// tenant's record holds, whatever the token claims.
func TestAuthorization_RoleComesFromTheTenantsRecord(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-roles").ID
	viewer := helpers.CreateTestUser(t, h.admin, tenantID, "viewer@authz-roles.test", "viewer")
	editor := helpers.CreateTestUser(t, h.admin, tenantID, "editor@authz-roles.test", "editor")
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@authz-roles.test", "admin")

	promote := "/api/v1/users/" + editor.ID + "/role"

	status, body := h.send(t, http.MethodPost, promote, tokenFor(t, tenantID, viewer.ID, "viewer"), `{"role":"admin"}`)
	assert.Equal(t, http.StatusForbidden, status, "a viewer granted a role: %s", body)

	// The claim says admin; the record says viewer. A token issued before a demotion looks
	// exactly like this.
	status, body = h.send(t, http.MethodPost, promote, tokenFor(t, tenantID, viewer.ID, "admin"), `{"role":"admin"}`)
	assert.Equal(t, http.StatusForbidden, status, "a claimed role outranked the record: %s", body)
	assert.Equal(t, "editor", h.roleOf(t, editor.ID), "the role changed")

	status, body = h.send(t, http.MethodPost, promote, tokenFor(t, tenantID, admin.ID, "admin"), `{"role":"owner"}`)
	assert.Equal(t, http.StatusForbidden, status, "an administrator granted the owner role: %s", body)

	status, body = h.send(t, http.MethodPost, promote, tokenFor(t, tenantID, admin.ID, "admin"), `{"role":"admin"}`)
	assert.Equal(t, http.StatusOK, status, "an administrator could not grant a role: %s", body)
	assert.Equal(t, "admin", h.roleOf(t, editor.ID))
}

// TestAuthorization_PasswordBelongsToTheSubject: a password change asks for the current
// password, and only the subject has it. Nobody else's role opens it.
func TestAuthorization_PasswordBelongsToTheSubject(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-password").ID
	viewer := helpers.CreateTestUser(t, h.admin, tenantID, "viewer@authz-password.test", "viewer")
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@authz-password.test", "admin")

	const change = `{"current_password":"password123","new_password":"another-password"}`

	status, body := h.send(t, http.MethodPost, "/api/v1/users/"+admin.ID+"/password",
		tokenFor(t, tenantID, viewer.ID, "viewer"), change)
	assert.Equal(t, http.StatusForbidden, status, "a viewer changed an administrator's password: %s", body)

	status, body = h.send(t, http.MethodPost, "/api/v1/users/"+viewer.ID+"/password",
		tokenFor(t, tenantID, admin.ID, "admin"), change)
	assert.Equal(t, http.StatusForbidden, status, "an administrator changed another user's password: %s", body)

	status, body = h.send(t, http.MethodPost, "/api/v1/users/"+viewer.ID+"/password",
		tokenFor(t, tenantID, viewer.ID, "viewer"), change)
	assert.Equal(t, http.StatusOK, status, "the subject could not change its own password: %s", body)
}

// TestAuthorization_AccessEndsWithTheMembership: a token outlives the membership it was
// issued for. Suspending or deleting the user ends access now, not when the token expires,
// and that includes the provisioning endpoint registered outside setupRoutes.
func TestAuthorization_AccessEndsWithTheMembership(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-revoked").ID
	other := helpers.CreateTestTenant(t, h.admin, "authz-revoked-other").ID
	member := helpers.CreateTestUser(t, h.admin, tenantID, "member@authz-revoked.test", "admin")
	outsider := helpers.CreateTestUser(t, h.admin, other, "outsider@authz-revoked.test", "admin")

	token := tokenFor(t, tenantID, member.ID, "admin")
	const provision = `{"name":"Another","slug":"authz-another","email":"another@example.com"}`

	status, body := h.send(t, http.MethodGet, "/api/v1/users", token, "")
	require.Equal(t, http.StatusOK, status, "an active member was refused: %s", body)

	_, err := h.admin.Exec(`UPDATE users SET status = 'suspended' WHERE id = $1`, member.ID)
	require.NoError(t, err)

	status, body = h.send(t, http.MethodGet, "/api/v1/users", token, "")
	assert.Equal(t, http.StatusForbidden, status, "a suspended member kept access: %s", body)

	status, body = h.send(t, http.MethodPost, "/api/v1/tenants", token, provision)
	assert.Equal(t, http.StatusForbidden, status, "a suspended member could create a tenant: %s", body)

	_, err = h.admin.Exec(`UPDATE users SET status = 'active', deleted_at = NOW() WHERE id = $1`, member.ID)
	require.NoError(t, err)

	status, body = h.send(t, http.MethodGet, "/api/v1/users", token, "")
	assert.Equal(t, http.StatusForbidden, status, "a deleted member kept access: %s", body)

	// A real subject of another tenant, carrying a genuine token that names this one.
	status, body = h.send(t, http.MethodGet, "/api/v1/users", tokenFor(t, tenantID, outsider.ID, "admin"), "")
	assert.Equal(t, http.StatusForbidden, status, "a subject of another tenant was let in: %s", body)
}
