package app

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/test/helpers"
)

// These tests run the installation endpoints over the application's own database role,
// which does not bypass row-level security. A superuser would pass every one of them with
// the tenant scope missing.

const (
	tenantModulesPath = "/api/v1/registry/tenant/modules"
	tenantToolsPath   = "/api/v1/registry/tenant/tools"
)

// tenantWithRole creates a tenant and one member of it holding role, and returns the
// tenant's id and a token for that member.
func tenantWithRole(t *testing.T, h *authzHarness, slug, role string) (string, string) {
	t.Helper()

	tenantID := helpers.CreateTestTenant(t, h.admin, slug).ID
	member := helpers.CreateTestUser(t, h.admin, tenantID, role+"@"+slug+".test", role)
	return tenantID, tokenFor(t, tenantID, member.ID, role)
}

// insertCountedModule adds a sparse module whose install count starts at zero. The trigger
// adds to the count it finds, and a NULL count stays NULL.
func insertCountedModule(t *testing.T, db *sql.DB, slug string) string {
	t.Helper()

	id := insertSparseModule(t, db, slug)
	_, err := db.Exec(`UPDATE modules SET install_count = 0 WHERE id = $1`, id)
	require.NoError(t, err)
	return id
}

// insertCountedTool is insertCountedModule for the tool catalogue.
func insertCountedTool(t *testing.T, db *sql.DB, slug string) string {
	t.Helper()

	id := insertSparseTool(t, db, slug)
	_, err := db.Exec(`UPDATE tools SET install_count = 0 WHERE id = $1`, id)
	require.NoError(t, err)
	return id
}

// installCount reads a catalogue entry's install count, which a trigger keeps.
func installCount(t *testing.T, db *sql.DB, table, id string) int {
	t.Helper()

	var count int
	require.NoError(t, db.QueryRow("SELECT install_count FROM "+table+" WHERE id = $1", id).Scan(&count))
	return count
}

// expect sends a request and fails the test unless the status is want, returning the body.
func expect(t *testing.T, h *authzHarness, token, method, path, body string, want int) string {
	t.Helper()

	status, got := h.send(t, method, path, token, body)
	require.Equalf(t, want, status, "%s %s: %s", method, path, got)
	return got
}

func TestTenantModules_FollowTheirLifecycle(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertCountedModule(t, h.admin, "lifecycle-module")
	tenantID, token := tenantWithRole(t, h, "lifecycle", "admin")
	install := `{"module_id":"` + moduleID + `","auto_activate":true}`
	module := tenantModulesPath + "/" + moduleID

	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", install, http.StatusCreated)
	assert.Contains(t, expect(t, h, token, http.MethodGet, tenantModulesPath, "", http.StatusOK), moduleID)
	assert.Equal(t, 1, installCount(t, h.admin, "modules", moduleID))

	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", install, http.StatusConflict)

	expect(t, h, token, http.MethodPost, module+"/deactivate", "", http.StatusOK)
	expect(t, h, token, http.MethodPost, module+"/deactivate", "", http.StatusConflict)
	assert.NotContains(t, expect(t, h, token, http.MethodGet, tenantModulesPath, "", http.StatusOK), moduleID)
	assert.Equal(t, 0, installCount(t, h.admin, "modules", moduleID))

	expect(t, h, token, http.MethodPost, module+"/activate", "", http.StatusOK)
	assert.Equal(t, 1, installCount(t, h.admin, "modules", moduleID))

	expect(t, h, token, http.MethodDelete, module, "", http.StatusNoContent)
	expect(t, h, token, http.MethodDelete, module, "", http.StatusNotFound)
	assert.NotContains(t, expect(t, h, token, http.MethodGet, tenantModulesPath, "", http.StatusOK), moduleID)
	assert.Equal(t, 0, installCount(t, h.admin, "modules", moduleID), "an uninstall while active is uncounted")

	// A module the tenant removed installs again, and its setup can be finished by hand.
	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", `{"module_id":"`+moduleID+`"}`, http.StatusCreated)
	expect(t, h, token, http.MethodPost, module+"/complete-setup", "", http.StatusOK)
	assert.Contains(t, expect(t, h, token, http.MethodGet, tenantModulesPath, "", http.StatusOK), moduleID)
	assert.Equal(t, 1, installCount(t, h.admin, "modules", moduleID))

	var rows int
	require.NoError(t, h.admin.QueryRow(
		`SELECT count(*) FROM tenant_modules WHERE tenant_id = $1 AND module_id = $2`, tenantID, moduleID,
	).Scan(&rows))
	assert.Equal(t, 1, rows, "a reinstall brings the tenant's row back instead of adding one")
}

func TestTenantTools_FollowTheirLifecycle(t *testing.T) {
	h := newAuthzHarness(t)

	toolID := insertCountedTool(t, h.admin, "lifecycle-tool")
	_, token := tenantWithRole(t, h, "tool-lifecycle", "admin")
	install := `{"tool_id":"` + toolID + `","auto_activate":true}`
	tool := tenantToolsPath + "/" + toolID

	expect(t, h, token, http.MethodPost, tenantToolsPath+"/install", install, http.StatusCreated)
	assert.Contains(t, expect(t, h, token, http.MethodGet, tenantToolsPath, "", http.StatusOK), toolID)
	assert.Equal(t, 1, installCount(t, h.admin, "tools", toolID))

	expect(t, h, token, http.MethodPost, tenantToolsPath+"/install", install, http.StatusConflict)
	expect(t, h, token, http.MethodPost, tool+"/deactivate", "", http.StatusOK)
	expect(t, h, token, http.MethodPost, tool+"/activate", "", http.StatusOK)
	expect(t, h, token, http.MethodPost, tool+"/complete-setup", "", http.StatusOK)

	expect(t, h, token, http.MethodDelete, tool, "", http.StatusNoContent)
	assert.NotContains(t, expect(t, h, token, http.MethodGet, tenantToolsPath, "", http.StatusOK), toolID)
	assert.Equal(t, 0, installCount(t, h.admin, "tools", toolID))

	expect(t, h, token, http.MethodPost, tenantToolsPath+"/install", install, http.StatusCreated)
	assert.Equal(t, 1, installCount(t, h.admin, "tools", toolID))
}

func TestTenantModules_WaitForTheirRequiredDependencies(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertCountedModule(t, h.admin, "dependent-module")
	baseID := insertCountedModule(t, h.admin, "base-module")
	_, err := h.admin.Exec(`
		INSERT INTO module_dependencies (module_id, depends_on_module_id, dependency_type)
		VALUES ($1, $2, 'required')`, moduleID, baseID)
	require.NoError(t, err)

	_, token := tenantWithRole(t, h, "dependent", "admin")
	install := func(id string) string { return `{"module_id":"` + id + `","auto_activate":true}` }

	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", install(moduleID), http.StatusConflict)
	assert.Contains(t,
		expect(t, h, token, http.MethodGet, tenantModulesPath+"/"+moduleID+"/dependencies", "", http.StatusOK),
		baseID, "the check names the module that is missing")

	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", install(baseID), http.StatusCreated)
	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", install(moduleID), http.StatusCreated)
}

// TestAudit_RecordsInstallations: each change an administrator makes to an installation
// leaves one line naming who made it and what it was made to, and the tenant's history
// shows it.
func TestAudit_RecordsInstallations(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertCountedModule(t, h.admin, "audited-module")
	toolID := insertCountedTool(t, h.admin, "audited-tool")
	tenantID := helpers.CreateTestTenant(t, h.admin, "audited").ID
	admin := helpers.CreateTestUser(t, h.admin, tenantID, "admin@audited.test", "admin")
	token := tokenFor(t, tenantID, admin.ID, "admin")
	module := tenantModulesPath + "/" + moduleID

	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", `{"module_id":"`+moduleID+`","auto_activate":true}`, http.StatusCreated)
	expect(t, h, token, http.MethodPost, module+"/deactivate", "", http.StatusOK)
	expect(t, h, token, http.MethodPost, module+"/activate", "", http.StatusOK)
	expect(t, h, token, http.MethodDelete, module, "", http.StatusNoContent)
	expect(t, h, token, http.MethodPost, tenantToolsPath+"/install", `{"tool_id":"`+toolID+`"}`, http.StatusCreated)
	expect(t, h, token, http.MethodPost, tenantToolsPath+"/"+toolID+"/complete-setup", "", http.StatusOK)

	rows, err := h.admin.Query(`
		SELECT action, actor_id::text, subject_type, subject_id::text, metadata::text
		FROM audit_log WHERE tenant_id = $1 ORDER BY created_at`, tenantID)
	require.NoError(t, err)
	defer rows.Close()

	subjects := map[string]struct{ id, code string }{
		"module": {moduleID, "AUDITED_MODULE"},
		"tool":   {toolID, "AUDITED_TOOL"},
	}

	var actions []string
	for rows.Next() {
		var action, actor, subjectType, subjectID, metadata string
		require.NoError(t, rows.Scan(&action, &actor, &subjectType, &subjectID, &metadata))
		actions = append(actions, action)

		assert.Equalf(t, admin.ID, actor, "%s does not say who made the change", action)
		assert.Equalf(t, subjects[subjectType].id, subjectID, "%s names the wrong subject", action)
		assert.Containsf(t, metadata, subjects[subjectType].code, "%s cannot be read without the catalogue", action)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{
		"module.installed", "module.activated", "module.deactivated", "module.activated", "module.uninstalled",
		"tool.installed", "tool.setup_completed",
	}, actions)

	assert.Contains(t, expect(t, h, token, http.MethodGet, "/api/v1/audit", "", http.StatusOK), "module.uninstalled")
}

// TestAudit_AnInstallationWithoutItsRecordDoesNotStand takes the right to write the trail
// away from the application's role. The install must fail whole: an installation nobody
// can account for is the state rule 7 exists to rule out.
func TestAudit_AnInstallationWithoutItsRecordDoesNotStand(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertCountedModule(t, h.admin, "unrecorded-module")
	tenantID, token := tenantWithRole(t, h, "unrecorded", "admin")

	var owner string
	require.NoError(t, h.admin.QueryRow(`SELECT tableowner FROM pg_tables WHERE tablename = 'audit_log'`).Scan(&owner))
	_, err := h.admin.Exec(`REVOKE INSERT ON audit_log FROM "` + owner + `"`)
	require.NoError(t, err)

	expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", `{"module_id":"`+moduleID+`","auto_activate":true}`, http.StatusInternalServerError)

	var rows int
	require.NoError(t, h.admin.QueryRow(`SELECT count(*) FROM tenant_modules WHERE tenant_id = $1`, tenantID).Scan(&rows))
	assert.Zero(t, rows, "the installation stood without its record")
	assert.Zero(t, installCount(t, h.admin, "modules", moduleID))
}

func TestTenantModules_BelongToOneTenant(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertCountedModule(t, h.admin, "shared-module")
	_, ownerToken := tenantWithRole(t, h, "first", "admin")
	_, otherToken := tenantWithRole(t, h, "second", "admin")
	install := `{"module_id":"` + moduleID + `","auto_activate":true}`
	module := tenantModulesPath + "/" + moduleID

	expect(t, h, ownerToken, http.MethodPost, tenantModulesPath+"/install", install, http.StatusCreated)

	// The other tenant neither sees the installation nor reaches it.
	assert.NotContains(t, expect(t, h, otherToken, http.MethodGet, tenantModulesPath, "", http.StatusOK), moduleID)
	expect(t, h, otherToken, http.MethodPost, module+"/deactivate", "", http.StatusNotFound)
	expect(t, h, otherToken, http.MethodPost, module+"/complete-setup", "", http.StatusNotFound)
	expect(t, h, otherToken, http.MethodDelete, module, "", http.StatusNotFound)
	assert.Contains(t, expect(t, h, ownerToken, http.MethodGet, tenantModulesPath, "", http.StatusOK), moduleID)

	// Its own installation of the same module is its own.
	expect(t, h, otherToken, http.MethodPost, tenantModulesPath+"/install", install, http.StatusCreated)
	assert.Equal(t, 2, installCount(t, h.admin, "modules", moduleID))
}

func TestTenantModules_AreChangedOnlyByAdministrators(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertCountedModule(t, h.admin, "guarded-module")
	_, editorToken := tenantWithRole(t, h, "editors", "editor")

	expect(t, h, editorToken, http.MethodPost, tenantModulesPath+"/install", `{"module_id":"`+moduleID+`"}`, http.StatusForbidden)
	expect(t, h, editorToken, http.MethodGet, tenantModulesPath, "", http.StatusOK)
}

func TestTenantModules_WhatDoesNotExistIsNotFound(t *testing.T) {
	h := newAuthzHarness(t)

	_, token := tenantWithRole(t, h, "unknown", "admin")

	for _, body := range []string{`{"module_id":"` + uuid.NewString() + `"}`, `{"module_id":"not-a-uuid"}`} {
		expect(t, h, token, http.MethodPost, tenantModulesPath+"/install", body, http.StatusNotFound)
	}
	expect(t, h, token, http.MethodPost, tenantToolsPath+"/install", `{"tool_id":"`+uuid.NewString()+`"}`, http.StatusNotFound)

	for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
		expect(t, h, token, http.MethodPost, tenantModulesPath+"/"+id+"/activate", "", http.StatusNotFound)
		expect(t, h, token, http.MethodPost, tenantToolsPath+"/"+id+"/activate", "", http.StatusNotFound)
	}
	expect(t, h, token, http.MethodGet, tenantModulesPath+"/not-a-uuid/dependencies", "", http.StatusNotFound)
	expect(t, h, token, http.MethodGet, tenantToolsPath+"/not-a-uuid/dependencies", "", http.StatusNotFound)
}
