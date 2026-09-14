package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The catalogue entries in these tests are written here rather than taken from the seed
// migration, so each test states which columns are empty instead of inheriting whatever
// the seed happens to fill in.

// insertSparseModule adds a public, active module with every optional column NULL,
// including those with a default. A default applies only when an insert leaves the column
// out, and a row written by another tool need not.
func insertSparseModule(t *testing.T, db *sql.DB, slug string) string {
	t.Helper()

	id := uuid.NewString()
	_, err := db.Exec(`
		INSERT INTO modules (id, name, slug, code, display_name, description, category,
			base_price, currency, features, capabilities, requires_database, default_limits,
			metadata, install_count, rating, review_count)
		VALUES ($1, $2::text, $2::text, upper(replace($2::text, '-', '_')), $2::text, 'nothing optional filled in', 'other',
			NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)`, id, slug)
	require.NoError(t, err)
	return id
}

// insertSparseTool is insertSparseModule for the tool catalogue.
func insertSparseTool(t *testing.T, db *sql.DB, slug string) string {
	t.Helper()

	id := uuid.NewString()
	_, err := db.Exec(`
		INSERT INTO tools (id, name, slug, code, display_name, description, category,
			base_price, currency, features, capabilities, requires_api_keys, default_limits,
			rate_limits, default_configuration, metadata, install_count, rating, review_count)
		VALUES ($1, $2::text, $2::text, upper(replace($2::text, '-', '_')), $2::text, 'nothing optional filled in', 'other',
			NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)`, id, slug)
	require.NoError(t, err)
	return id
}

func TestCatalog_ReadsEntriesWithEmptyOptionalColumns(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertSparseModule(t, h.admin, "sparse-module")
	toolID := insertSparseTool(t, h.admin, "sparse-tool")

	for _, path := range []string{
		"/api/v1/registry/modules/" + moduleID,
		"/api/v1/registry/modules/slug/sparse-module",
		"/api/v1/registry/tools/" + toolID,
		"/api/v1/registry/tools/slug/sparse-tool",
	} {
		status, body := h.send(t, http.MethodGet, path, "", "")
		assert.Equalf(t, http.StatusOK, status, "%s: %s", path, body)
	}

	// The lists decode the same rows through the same columns.
	for path, slug := range map[string]string{
		"/api/v1/registry/modules?per_page=100": "sparse-module",
		"/api/v1/registry/tools?per_page=100":   "sparse-tool",
	} {
		status, body := h.send(t, http.MethodGet, path, "", "")
		assert.Equalf(t, http.StatusOK, status, "%s: %s", path, body)
		assert.Containsf(t, body, `"`+slug+`"`, "%s leaves out the sparse entry", path)
	}
}

func TestCatalog_ReadsArraysAndDocumentsBack(t *testing.T) {
	h := newAuthzHarness(t)

	moduleID := insertSparseModule(t, h.admin, "filled-module")
	_, err := h.admin.Exec(`
		UPDATE modules SET tags = '{alpha,beta}', configuration_schema = '{"type": "object"}',
			min_platform_version = '0.2.0', created_by = $2
		WHERE id = $1`, moduleID, uuid.NewString())
	require.NoError(t, err)

	status, body := h.send(t, http.MethodGet, "/api/v1/registry/modules/"+moduleID, "", "")
	require.Equal(t, http.StatusOK, status, body)

	var detail struct {
		Data struct {
			Tags                []string        `json:"tags"`
			ConfigurationSchema json.RawMessage `json:"configuration_schema"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &detail))
	assert.Equal(t, []string{"alpha", "beta"}, detail.Data.Tags)
	assert.JSONEq(t, `{"type": "object"}`, string(detail.Data.ConfigurationSchema))
}

func TestCatalog_AnEntryThatIsNotThereIsNotFound(t *testing.T) {
	h := newAuthzHarness(t)

	for _, path := range []string{
		"/api/v1/registry/modules/" + uuid.NewString(),
		"/api/v1/registry/modules/not-a-uuid",
		"/api/v1/registry/modules/slug/nothing-here",
		"/api/v1/registry/tools/" + uuid.NewString(),
		"/api/v1/registry/tools/not-a-uuid",
		"/api/v1/registry/tools/slug/nothing-here",
	} {
		status, body := h.send(t, http.MethodGet, path, "", "")
		assert.Equalf(t, http.StatusNotFound, status, "%s: %s", path, body)
	}
}

// TestCatalog_TheSortKeyIsNeverSQL sends expressions as the sort key. The key goes into
// ORDER BY, where no bind parameter can stand in for it. 1/0 is the probe: a database that
// evaluates it fails the query, and one that never sees it answers in the default order.
func TestCatalog_TheSortKeyIsNeverSQL(t *testing.T) {
	h := newAuthzHarness(t)

	insertSparseModule(t, h.admin, "beta-module")
	insertSparseModule(t, h.admin, "alpha-module")

	for _, base := range []string{"/api/v1/registry/modules", "/api/v1/registry/tools"} {
		for _, key := range []string{"(SELECT 1/0)", "no_such_column", "name; SELECT 1"} {
			status, body := h.send(t, http.MethodGet, base+"?sort_by="+url.QueryEscape(key), "", "")
			assert.Equalf(t, http.StatusOK, status, "%s sort_by=%q: %s", base, key, body)
			assert.NotContainsf(t, body, "pq:", "%s sort_by=%q", base, key)
		}
	}

	// A documented key still sorts.
	status, body := h.send(t, http.MethodGet, "/api/v1/registry/modules?sort_by=name&sort_order=asc&per_page=100", "", "")
	require.Equal(t, http.StatusOK, status, body)
	alpha, beta := strings.Index(body, `"alpha-module"`), strings.Index(body, `"beta-module"`)
	require.True(t, alpha >= 0 && beta >= 0, body)
	assert.Less(t, alpha, beta)
}
