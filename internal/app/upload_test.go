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
// test stores. The handler writes under ./uploads, relative to this package.
func uploaderFor(t *testing.T, h *authzHarness, slug string) (string, string) {
	t.Helper()
	t.Cleanup(func() { _ = os.RemoveAll("./uploads") })

	tenantID := helpers.CreateTestTenant(t, h.admin, slug).ID
	editor := helpers.CreateTestUser(t, h.admin, tenantID, "editor@"+slug+".test", "editor")
	return tenantID, tokenFor(t, tenantID, editor.ID, "editor")
}

// TestUpload_StoresOnlyTheImagesItCanRecognise: the type is read from the file, not taken
// from the request. A page declared as a PNG was stored as .html and served back from this
// origin as text/html, where its script ran for anyone who opened the link.
func TestUpload_StoresOnlyTheImagesItCanRecognise(t *testing.T) {
	h := newAuthzHarness(t)
	_, token := uploaderFor(t, h, "upload-types")

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
	tenantID, token := uploaderFor(t, h, "upload-serving")

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
