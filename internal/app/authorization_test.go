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
	auditHandler "github.com/canakyuz/keystone/internal/handler/audit"
	authHandler "github.com/canakyuz/keystone/internal/handler/auth"
	operationHandler "github.com/canakyuz/keystone/internal/handler/operation"
	registryHandler "github.com/canakyuz/keystone/internal/handler/registry"
	tenantHandler "github.com/canakyuz/keystone/internal/handler/tenant"
	uploadHandler "github.com/canakyuz/keystone/internal/handler/upload"
	userHandler "github.com/canakyuz/keystone/internal/handler/user"
	webhookHandler "github.com/canakyuz/keystone/internal/handler/webhook"
	"github.com/canakyuz/keystone/internal/middleware"
	auditRepo "github.com/canakyuz/keystone/internal/repository/audit"
	operationRepo "github.com/canakyuz/keystone/internal/repository/operation"
	platformRepo "github.com/canakyuz/keystone/internal/repository/platform"
	registryRepo "github.com/canakyuz/keystone/internal/repository/registry"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	uploadRepo "github.com/canakyuz/keystone/internal/repository/upload"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	webhookRepo "github.com/canakyuz/keystone/internal/repository/webhook"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
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
// and says nothing about whether the application applies it. The application's error
// handler is mounted too, so a test reads the body a client would.
func newAuthzHarness(t *testing.T) *authzHarness {
	t.Helper()

	admin := helpers.SetupTestDB(t)
	appDB := helpers.SetupAppRoleDB(t, admin)

	middleware.AllowUntrustedTenantSource(false)

	cfg := &config.Config{}
	cfg.Auth.JWTSecret = authzSecret

	log := logger.Default()
	trail := auditRepo.New()
	tenants := tenantRepo.NewPostgresRepository(appDB).WithAudit(trail)
	users := userRepo.NewPostgresRepository(appDB, database.NewTenantConnectionManager(appDB, nil)).WithAudit(trail)
	userService := userUsecase.NewService(users, validator.New(), log, authzSecret)
	membership := middleware.Membership(users)

	// The registry, built the way the composition root builds it, so the route table under
	// test has no nil handler behind any route.
	modules := registryRepo.NewModuleRepository(appDB)
	tools := registryRepo.NewToolRepository(appDB)
	tenantModules := registryRepo.NewTenantModuleRepository(appDB, trail)
	tenantTools := registryRepo.NewTenantToolRepository(appDB, trail)
	dependencies := registryService.NewDependencyCheckerService(appDB, modules, tools, tenantModules, tenantTools)
	activation := registryService.NewTenantActivationService(modules, tools, tenantModules, tenantTools, dependencies)
	platformOnly := middleware.PlatformOnly(platformRepo.New(appDB))

	app := fiber.New(fiber.Config{DisableStartupMessage: true, ErrorHandler: customErrorHandler})
	planRateLimit := func(c *fiber.Ctx) error { return c.Next() }

	registerOperationRoutes(app, authzSecret, membership, platformOnly,
		operationHandler.New(operationRepo.New(appDB), log))

	setupRoutes(app, cfg,
		authHandler.NewHandler(userService),
		auditHandler.NewHandler(auditRepo.NewReader(appDB)),
		webhookHandler.NewHandler(webhookRepo.New(appDB, trail)),
		tenantHandler.NewHandler(tenantUsecase.NewService(tenants, validator.New(), log, nil)),
		userHandler.NewHandler(userService),
		uploadHandler.NewHandler(log, uploadRepo.New(appDB, trail)),
		registryHandler.NewModuleCatalogHandler(registryService.NewModuleCatalogService(modules)),
		registryHandler.NewToolCatalogHandler(registryService.NewToolCatalogService(tools)),
		registryHandler.NewActivationHandler(activation, dependencies),
		middleware.TenantContextMiddleware(middleware.NewTenantSchemaCache(nil, appDB, nil, nil)),
		middleware.TenantScope(tenants, database.NewTenantManager(appDB)),
		planRateLimit,
		membership,
		platformOnly,
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

// grantPlatform records the subject in platform_operators, the way an operator would be
// given the permission outside the API.
func (h *authzHarness) grantPlatform(t *testing.T, userID string) {
	t.Helper()

	_, err := h.admin.Exec(
		`INSERT INTO platform_operators (user_id, note) VALUES ($1, 'authorization test')`, userID)
	require.NoError(t, err)
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

// TestAuthorization_PlatformRoutesNeedTheGrant: listing, counting and lifecycle changes act
// across tenants. No tenant role opens them, and the grant in platform_operators does.
func TestAuthorization_PlatformRoutesNeedTheGrant(t *testing.T) {
	h := newAuthzHarness(t)

	own := helpers.CreateTestTenant(t, h.admin, "authz-platform").ID
	other := helpers.CreateTestTenant(t, h.admin, "authz-platform-other").ID
	owner := helpers.CreateTestUser(t, h.admin, own, "owner@authz-platform.test", "owner")
	token := tokenFor(t, own, owner.ID, "owner")

	const provision = `{"name":"Operator Made","slug":"authz-operator-made","email":"made@example.com"}`

	platform := []probe{
		{http.MethodGet, "/api/v1/tenants", ""},
		{http.MethodGet, "/api/v1/tenants/stats", ""},
		{http.MethodGet, "/api/v1/tenants/slug/authz-platform", ""},
		{http.MethodPost, "/api/v1/tenants/" + own + "/suspend", `{"reason":"x"}`},
		{http.MethodPost, "/api/v1/tenants/" + own + "/activate", ""},
		{http.MethodPost, "/api/v1/tenants/" + own + "/upgrade", `{"plan":"enterprise"}`},
		{http.MethodPost, "/api/v1/tenants", provision},
	}

	// An owner of a tenant is nobody at the platform level.
	for _, p := range platform {
		status, body := h.send(t, p.method, p.path, token, p.body)
		assert.Equal(t, http.StatusForbidden, status, "%s %s: %s", p.method, p.path, body)
	}

	h.grantPlatform(t, owner.ID)

	status, body := h.send(t, http.MethodGet, "/api/v1/tenants", token, "")
	assert.Equal(t, http.StatusOK, status, "an operator could not list the tenants: %s", body)

	// Across tenants, which is the whole point of the permission: this is not the
	// operator's own tenant.
	status, body = h.send(t, http.MethodPost, "/api/v1/tenants/"+other+"/suspend", token, `{"reason":"checking"}`)
	assert.Equal(t, http.StatusOK, status, "an operator could not suspend another tenant: %s", body)

	var otherStatus string
	require.NoError(t, h.admin.QueryRow(`SELECT status FROM tenants WHERE id = $1`, other).Scan(&otherStatus))
	assert.Equal(t, "suspended", otherStatus)

	status, body = h.send(t, http.MethodPost, "/api/v1/tenants", token, provision)
	assert.Equal(t, http.StatusAccepted, status, "an operator could not create a tenant: %s", body)
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

// TestAudit_RecordsWhoChangedWhat: a change made through the API leaves a record in the
// transaction that made it, with the actor read from the request rather than passed in by
// the call site.
func TestAudit_RecordsWhoChangedWhat(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-trail").ID
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@authz-trail.test", "admin")
	editor := helpers.CreateTestUser(t, h.admin, tenantID, "editor@authz-trail.test", "editor")
	token := tokenFor(t, tenantID, admin.ID, "admin")

	status, body := h.send(t, http.MethodPost, "/api/v1/users/"+editor.ID+"/role", token, `{"role":"viewer"}`)
	require.Equal(t, http.StatusOK, status, "the role change was refused: %s", body)

	status, body = h.send(t, http.MethodPost, "/api/v1/users/"+editor.ID+"/suspend", token, `{"reason":"left the team"}`)
	require.Equal(t, http.StatusOK, status, "the suspension was refused: %s", body)

	var actor, subject, metadata string
	require.NoError(t, h.admin.QueryRow(`
		SELECT actor_id::text, subject_id::text, metadata::text
		FROM audit_log
		WHERE tenant_id = $1 AND action = 'user.role_changed'`, tenantID,
	).Scan(&actor, &subject, &metadata))

	assert.Equal(t, admin.ID, actor, "the trail does not say who made the change")
	assert.Equal(t, editor.ID, subject)
	assert.Contains(t, metadata, "editor", "the trail does not say what the role was")
	assert.Contains(t, metadata, "viewer", "the trail does not say what the role became")

	var suspensions int
	require.NoError(t, h.admin.QueryRow(
		`SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'user.suspended'`, tenantID,
	).Scan(&suspensions))
	assert.Equal(t, 1, suspensions, "the suspension left no record")

	// The operator path records the tenant itself as the subject.
	h.grantPlatform(t, admin.ID)

	status, body = h.send(t, http.MethodPost, "/api/v1/tenants/"+tenantID+"/suspend", token, `{"reason":"unpaid"}`)
	require.Equal(t, http.StatusOK, status, "the tenant suspension was refused: %s", body)

	var tenantActor, tenantAction string
	require.NoError(t, h.admin.QueryRow(`
		SELECT actor_id::text, action
		FROM audit_log
		WHERE tenant_id = $1 AND subject_type = 'tenant'`, tenantID,
	).Scan(&tenantActor, &tenantAction))

	assert.Equal(t, admin.ID, tenantActor)
	assert.Equal(t, "tenant.suspended", tenantAction)
}

// TestAudit_IsReadOnlyByThisTenantsAdministrators: the trail is readable, by the people whose
// tenant it describes and by nobody else.
func TestAudit_IsReadOnlyByThisTenantsAdministrators(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-trail-read").ID
	other := helpers.CreateTestTenant(t, h.admin, "authz-trail-other").ID
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@authz-trail-read.test", "admin")
	editor := helpers.CreateTestUser(t, h.admin, tenantID, "editor@authz-trail-read.test", "editor")
	viewer := helpers.CreateTestUser(t, h.admin, tenantID, "viewer@authz-trail-read.test", "viewer")
	outsider := helpers.CreateTestUser(t, h.admin, other, "admin@authz-trail-other.test", "admin")

	adminToken := tokenFor(t, tenantID, admin.ID, "admin")

	status, body := h.send(t, http.MethodPost, "/api/v1/users/"+editor.ID+"/role", adminToken, `{"role":"viewer"}`)
	require.Equal(t, http.StatusOK, status, "the role change was refused: %s", body)

	status, body = h.send(t, http.MethodGet, "/api/v1/audit", adminToken, "")
	require.Equal(t, http.StatusOK, status, "an administrator could not read the trail: %s", body)
	assert.Contains(t, body, "user.role_changed")
	assert.Contains(t, body, admin.Email, "the trail does not name the actor")

	status, body = h.send(t, http.MethodGet, "/api/v1/audit", tokenFor(t, tenantID, viewer.ID, "viewer"), "")
	assert.Equal(t, http.StatusForbidden, status, "a viewer read the trail: %s", body)

	// Another tenant's administrator sees their own history, which here is empty, and not
	// this one's.
	status, body = h.send(t, http.MethodGet, "/api/v1/audit", tokenFor(t, other, outsider.ID, "admin"), "")
	require.Equal(t, http.StatusOK, status, body)
	assert.NotContains(t, body, "user.role_changed", "another tenant's trail leaked")
	assert.NotContains(t, body, editor.ID)
}

// TestAudit_RecordsMembersAndSettings: adding a member, changing a password, editing a
// profile and editing the tenant each leave a line, and none of those lines holds the
// secret or the personal data that changed. The trail says which fields changed, not what
// they became.
func TestAudit_RecordsMembersAndSettings(t *testing.T) {
	h := newAuthzHarness(t)

	tenantID := helpers.CreateTestTenant(t, h.admin, "authz-trail-more").ID
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@authz-trail-more.test", "admin")
	member := helpers.CreateTestUser(t, h.admin, tenantID, "member@authz-trail-more.test", "viewer")
	adminToken := tokenFor(t, tenantID, admin.ID, "admin")
	memberToken := tokenFor(t, tenantID, member.ID, "viewer")

	const newPassword = "a-new-password-9"

	for _, step := range []struct{ token, method, path, body string }{
		{adminToken, http.MethodPost, "/api/v1/users",
			`{"email":"new@authz-trail-more.test","password":"password123","first_name":"New","last_name":"Member","role":"viewer"}`},
		{memberToken, http.MethodPost, "/api/v1/users/" + member.ID + "/password",
			`{"current_password":"password123","new_password":"` + newPassword + `"}`},
		{adminToken, http.MethodPatch, "/api/v1/users/" + member.ID, `{"first_name":"Renamed","phone":"+44 20 0000 0000"}`},
		{adminToken, http.MethodPatch, "/api/v1/tenants/" + tenantID, `{"name":"Renamed Tenant"}`},
		{adminToken, http.MethodPatch, "/api/v1/tenants/" + tenantID + "/branding", `{"primary_color":"#112233"}`},
	} {
		status, body := h.send(t, step.method, step.path, step.token, step.body)
		require.Truef(t, status >= 200 && status < 300, "%s %s: %d %s", step.method, step.path, status, body)
	}

	rows, err := h.admin.Query(
		`SELECT action, coalesce(actor_id::text, ''), metadata::text FROM audit_log WHERE tenant_id = $1`, tenantID)
	require.NoError(t, err)
	defer rows.Close()

	actors := map[string]string{}
	var trail strings.Builder

	for rows.Next() {
		var action, actor, metadata string
		require.NoError(t, rows.Scan(&action, &actor, &metadata))
		actors[action] = actor
		trail.WriteString(metadata)
	}
	require.NoError(t, rows.Err())

	for action, actor := range map[string]string{
		"user.created":            admin.ID,
		"user.password_changed":   member.ID,
		"user.profile_updated":    admin.ID,
		"tenant.updated":          admin.ID,
		"tenant.branding_updated": admin.ID,
	} {
		assert.Equalf(t, actor, actors[action], "%s is missing, or names the wrong actor", action)
	}

	assert.NotContains(t, trail.String(), newPassword, "a password reached the trail")
	assert.NotContains(t, trail.String(), "password123", "a password reached the trail")
	assert.NotContains(t, trail.String(), "+44 20", "a phone number reached the trail")
}
