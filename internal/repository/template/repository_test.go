package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The test runs with the package directory as its working directory, where the old
// templates/tenants path did not exist; it passes only because the SQL is embedded.
func TestEmbeddedTemplatesResolveWithoutTheWorkingDirectory(t *testing.T) {
	repo := NewRepository()

	pro, err := repo.GetTemplateByPlan(context.Background(), "pro")
	require.NoError(t, err)
	assert.NotEmpty(t, pro)

	fallback, err := repo.GetTemplateByPlan(context.Background(), "no-such-plan")
	require.NoError(t, err)
	def, err := repo.GetTemplateByPlan(context.Background(), "default")
	require.NoError(t, err)
	assert.Equal(t, def, fallback, "an unknown plan falls back to default.sql")
}
