package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// documentedRoots are the directories held to the rule. The reference application under
// examples/ is not among them; see docs/ROADMAP.md, phase 5.
var documentedRoots = []string{"internal", "pkg", "cmd", "test/helpers"}

// TestExportedSymbolsAreDocumented holds the control plane to a doc comment on every
// exported function, method, type, constant and variable.
//
// It parses source rather than loading packages: whether a comment is there is a property
// of the file, and the parser answers that without type-checking anything. Generated files
// are skipped, because their comments belong to the generator, and so are methods on
// unexported types, which godoc never shows. A comment on a const or var group covers the
// whole group, as it does in godoc.
func TestExportedSymbolsAreDocumented(t *testing.T) {
	var missing []string

	for _, root := range documentedRoots {
		err := filepath.WalkDir(filepath.Join("../..", root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			missing = append(missing, undocumented(t, path)...)

			return nil
		})
		require.NoError(t, err)
	}

	assert.Empty(t, missing, "exported symbols without a doc comment:\n%s", strings.Join(missing, "\n"))
}

// undocumented lists the exported declarations in one file that carry no doc comment.
func undocumented(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	require.NoError(t, err)

	if ast.IsGenerated(file) {
		return nil
	}

	var missing []string
	report := func(pos token.Pos, name string) {
		missing = append(missing, strings.TrimPrefix(fset.Position(pos).String(), "../../")+" "+name)
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Doc == nil && d.Name.IsExported() && receiverIsExported(d) {
				report(d.Pos(), d.Name.Name)
			}
		case *ast.GenDecl:
			if d.Doc != nil {
				continue
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Doc == nil && s.Name.IsExported() {
						report(s.Pos(), s.Name.Name)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if s.Doc == nil && name.IsExported() {
							report(name.Pos(), name.Name)
						}
					}
				}
			}
		}
	}

	return missing
}

// receiverIsExported reports whether a method's receiver type is exported. A plain
// function has no receiver and counts as exported.
func receiverIsExported(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return true
	}

	expr := fn.Recv.List[0].Type
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return e.IsExported()
		default:
			return true
		}
	}
}
