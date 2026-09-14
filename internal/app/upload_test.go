package app

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/test/helpers"
)

// pngBytes is the eight-byte PNG signature and a little padding: enough for the type to be
// read from the content.
var pngBytes = "\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 32)

var uploadedURL = regexp.MustCompile(`"url":"([^"]+)"`)

// sendFile posts one multipart file the way a browser would, with the name and declared
// type the test chooses.
func sendFile(t *testing.T, h *authzHarness, token, path, filename, declaredType, content string) (int, string) {
	t.Helper()

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="` + filename + `"`},
		"Content-Type":        {declaredType},
	})
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, form.Close())

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", form.FormDataContentType())

	resp, err := h.app.Test(req, 10_000)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// uploaderFor creates a tenant with one editor, who may upload images, and removes what the
// test stores. The handler writes under ./uploads, relative to this package. It returns the
// tenant's id, the editor's id and a token for the editor.
func uploaderFor(t *testing.T, h *authzHarness, slug string) (string, string, string) {
	t.Helper()
	t.Cleanup(func() { _ = os.RemoveAll("./uploads") })

	tenantID := helpers.CreateTestTenant(t, h.admin, slug).ID
	editor := helpers.CreateTestUser(t, h.admin, tenantID, "editor@"+slug+".test", "editor")
	return tenantID, editor.ID, tokenFor(t, tenantID, editor.ID, "editor")
}

// TestUpload_StoresOnlyTheImagesItCanRecognise: the type is read from the file, not taken
// from the request. A page declared as a PNG was stored as .html and served back from this
// origin as text/html, where its script ran for anyone who opened the link.
func TestUpload_StoresOnlyTheImagesItCanRecognise(t *testing.T) {
	h := newAuthzHarness(t)
	_, _, token := uploaderFor(t, h, "upload-types")

	for _, refused := range []struct{ filename, declared, content string }{
		{"page.html", "image/png", "<html><script>alert(document.domain)</script></html>"},
		{"vector.svg", "image/svg+xml", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`},
		{"notes.png", "image/png", "plain text with an image extension"},
	} {
		status, body := sendFile(t, h, token, "/api/v1/upload/image", refused.filename, refused.declared, refused.content)
		assert.Equalf(t, http.StatusBadRequest, status, "%s declared as %s was stored: %s", refused.filename, refused.declared, body)
	}

	// A real image keeps the extension of what it is, whatever it was called.
	status, body := sendFile(t, h, token, "/api/v1/upload/image", "page.html", "text/html", pngBytes)
	require.Equal(t, http.StatusOK, status, body)

	url := uploadedURL.FindStringSubmatch(body)
	require.NotNil(t, url, body)
	assert.True(t, strings.HasSuffix(url[1], ".png"), "stored as %s", url[1])
}

// TestUpload_ServesFilesAsPassiveContent: whatever is already on disk is served with the
// type it has and in a sandbox, so a file stored before the type check cannot run script
// with this origin's authority.
func TestUpload_ServesFilesAsPassiveContent(t *testing.T) {
	h := newAuthzHarness(t)
	tenantID, _, token := uploaderFor(t, h, "upload-serving")

	status, body := sendFile(t, h, token, "/api/v1/upload/image", "logo.png", "image/png", pngBytes)
	require.Equal(t, http.StatusOK, status, body)
	url := uploadedURL.FindStringSubmatch(body)
	require.NotNil(t, url, body)

	// A page left behind by the old handler.
	legacy := "./uploads/tenants/" + tenantID + "/images/legacy.html"
	require.NoError(t, os.WriteFile(legacy, []byte("<script>alert(1)</script>"), 0o644))

	for _, path := range []string{url[1], "/uploads/tenants/" + tenantID + "/images/legacy.html"} {
		resp, err := h.app.Test(httptest.NewRequest(http.MethodGet, path, nil), 10_000)
		require.NoError(t, err)
		resp.Body.Close()

		require.Equalf(t, http.StatusOK, resp.StatusCode, path)
		assert.Equalf(t, "nosniff", resp.Header.Get("X-Content-Type-Options"), path)
		assert.Containsf(t, resp.Header.Get("Content-Security-Policy"), "sandbox", path)
	}
}

// TestUpload_IsRecordedWithWhoStoredIt: a stored file has a row in its tenant and a line in
// the trail, both naming the member who stored it.
func TestUpload_IsRecordedWithWhoStoredIt(t *testing.T) {
	h := newAuthzHarness(t)
	tenantID, editorID, token := uploaderFor(t, h, "upload-record")

	status, body := sendFile(t, h, token, "/api/v1/upload/image", "photo of me.png", "image/png", pngBytes)
	require.Equal(t, http.StatusOK, status, body)
	url := uploadedURL.FindStringSubmatch(body)
	require.NotNil(t, url, body)

	var uploadID, category, contentType, uploadedBy string
	var size int64
	require.NoError(t, h.admin.QueryRow(`
		SELECT id::text, category, content_type, size_bytes, uploaded_by::text
		FROM uploads WHERE tenant_id = $1 AND path = $2`, tenantID, url[1],
	).Scan(&uploadID, &category, &contentType, &size, &uploadedBy))
	assert.Equal(t, "image", category)
	assert.Equal(t, "image/png", contentType)
	assert.Equal(t, int64(len(pngBytes)), size)
	assert.Equal(t, editorID, uploadedBy)

	var actor, metadata string
	require.NoError(t, h.admin.QueryRow(`
		SELECT actor_id::text, metadata::text FROM audit_log
		WHERE tenant_id = $1 AND action = 'upload.created' AND subject_id = $2`, tenantID, uploadID,
	).Scan(&actor, &metadata))
	assert.Equal(t, editorID, actor, "the trail does not say who stored the file")
	assert.NotContains(t, metadata, "photo of me", "the client's file name reached the append-only trail")
}

// TestUpload_AFileWithoutItsRecordIsNotKept takes the right to record uploads away from the
// application's role. The request must fail and leave nothing on disk.
func TestUpload_AFileWithoutItsRecordIsNotKept(t *testing.T) {
	h := newAuthzHarness(t)
	tenantID, _, token := uploaderFor(t, h, "upload-unrecorded")

	var owner string
	require.NoError(t, h.admin.QueryRow(`SELECT tableowner FROM pg_tables WHERE tablename = 'uploads'`).Scan(&owner))
	_, err := h.admin.Exec(`REVOKE INSERT ON uploads FROM "` + owner + `"`)
	require.NoError(t, err)

	status, body := sendFile(t, h, token, "/api/v1/upload/image", "logo.png", "image/png", pngBytes)
	require.Equal(t, http.StatusInternalServerError, status, body)
	assert.NotContains(t, body, "permission denied", "the database's refusal reached the client")

	entries, err := os.ReadDir("./uploads/tenants/" + tenantID + "/images")
	if !os.IsNotExist(err) {
		require.NoError(t, err)
		assert.Empty(t, entries, "a file stayed on disk with no record of it")
	}
}
