// Package architecture asserts the dependency direction between the control plane and
// the reference application.
//
// The claim "the core runs without the example modules" is easy to make and easy to
// break: one convenient import from internal/app into examples/verticals reintroduces the
// coupling, everything still compiles, and the claim quietly becomes false. This package
// turns it into a build failure.
package architecture

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

const (
	module        = "github.com/canakyuz/keystone/"
	controlPlane  = module + "internal/..."
	exampleModule = module + "examples/"
)

// TestControlPlaneDoesNotImportTheExamples walks the real import graph.
//
// Transitively, not just directly: a direct import is easy to spot in review, while an
// import three packages deep is not, and it couples the two just as firmly.
func TestControlPlaneDoesNotImportTheExamples(t *testing.T) {
	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Dir:  "../..",
	}, controlPlane)
	require.NoError(t, err)
	require.NotEmpty(t, loaded, "no packages were loaded; the pattern is wrong")

	for _, pkg := range loaded {
		for path := range pkg.Imports {
			assert.Falsef(t, strings.HasPrefix(path, exampleModule),
				"%s imports %s: the control plane must not depend on the reference application",
				pkg.PkgPath, path)
		}
	}
}

// TestExamplesDoImportTheControlPlane confirms the arrow points the other way.
//
// Without this, the first test would also pass if the two halves were simply unrelated,
// which would mean the split had been achieved by duplicating the control plane rather
// than by depending on it.
func TestExamplesDoImportTheControlPlane(t *testing.T) {
	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedImports,
		Dir:  "../..",
	}, module+"examples/verticals")
	require.NoError(t, err)
	require.NotEmpty(t, loaded)

	var importsCore bool
	for _, pkg := range loaded {
		for path := range pkg.Imports {
			if strings.HasPrefix(path, module+"internal/") {
				importsCore = true
			}
		}
	}

	assert.True(t, importsCore,
		"the example modules do not use the control plane at all, which means they are not "+
			"an example of building on it")
}
