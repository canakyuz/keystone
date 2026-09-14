package upload

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
	uploadrepo "github.com/canakyuz/keystone/internal/repository/upload"
	"github.com/canakyuz/keystone/test/helpers"
)

// TestSweeper_FindsOldFilesTheirTenantDidNotRecord runs the sweep over the application's
// own database role, so each tenant's records are read under that tenant's policy.
func TestSweeper_FindsOldFilesTheirTenantDidNotRecord(t *testing.T) {
	admin := helpers.SetupTestDB(t)
	appDB := helpers.SetupAppRoleDB(t, admin)
	root := t.TempDir()
	ctx := context.Background()

	first := helpers.CreateTestTenant(t, admin, "sweep-first").ID
	second := helpers.CreateTestTenant(t, admin, "sweep-second").ID
	old := time.Now().Add(-2 * time.Hour)

	// store writes a file as the handler lays it out and returns where it is served from.
	store := func(tenantDir, name string, modified time.Time) (string, string) {
		dir := filepath.Join(root, "tenants", tenantDir, "images")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		full := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(full, []byte("x"), 0o644))
		require.NoError(t, os.Chtimes(full, modified, modified))
		return "/uploads/tenants/" + tenantDir + "/images/" + name, full
	}
	record := func(tenantID, served string) {
		_, err := admin.Exec(`
			INSERT INTO uploads (tenant_id, category, path, content_type, size_bytes)
			VALUES ($1, 'image', $2, 'image/png', 1)`, tenantID, served)
		require.NoError(t, err)
	}

	recorded, recordedFile := store(first, "recorded.png", old)
	record(first, recorded)
	orphan, orphanFile := store(first, "orphan.png", old)
	_, inFlightFile := store(first, "in-flight.png", time.Now())
	// A row in the first tenant does not account for a file in the second tenant's directory.
	borrowed, borrowedFile := store(second, "borrowed.png", old)
	record(first, borrowed)
	// A directory that is not named by a tenant is not the sweep's.
	_, strayFile := store("not-a-tenant", "stray.png", old)

	records := uploadrepo.New(appDB, auditrepo.New())

	result, err := NewSweeper(root, records, false, nil).Sweep(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{orphan, borrowed}, result.Unrecorded)
	assert.Zero(t, result.Removed)
	assert.FileExists(t, orphanFile, "a reporting sweep removed a file")

	result, err = NewSweeper(root, records, true, nil).Sweep(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Removed)
	assert.NoFileExists(t, orphanFile)
	assert.NoFileExists(t, borrowedFile)
	assert.FileExists(t, recordedFile)
	assert.FileExists(t, inFlightFile, "a file whose record may still be committing was removed")
	assert.FileExists(t, strayFile)
}

func TestSweeper_HasNothingToDoWithoutAnUploadDirectory(t *testing.T) {
	result, err := NewSweeper(filepath.Join(t.TempDir(), "missing"), nil, true, nil).Sweep(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result.Unrecorded)
}
