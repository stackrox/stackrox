package all

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var whitelist = []string{"all", "utils", "validation", "metadatagetter"}

func TestAllPackagesAreImported(t *testing.T) {
	notifiersDirEntry, err := os.ReadDir("..")
	require.NoError(t, err, "failed to read notifiers directory")

	existingNotifiers := set.NewStringSet()
	for _, entry := range notifiersDirEntry {
		if !entry.IsDir() {
			continue
		}
		baseName := filepath.Base(entry.Name())

		if slices.Contains(whitelist, baseName) {
			continue
		}
		existingNotifiers.Add(baseName)
	}

	var allImports []*ast.ImportSpec
	f, err := parser.ParseFile(token.NewFileSet(), "all.go", nil, parser.ImportsOnly)
	require.NoError(t, err, "failed to parse all.go")

	allImports = append(allImports, f.Imports...)

	importedNotifiers := set.NewStringSet()
	for _, imp := range allImports {
		pkgName := strings.TrimSuffix(strings.TrimPrefix(imp.Path.Value, `"`), `"`)
		pkgBaseName := path.Base(pkgName)
		importedNotifiers.Add(pkgBaseName)
	}

	existingButNotImported := existingNotifiers.Difference(importedNotifiers)
	importedButNotExisting := importedNotifiers.Difference(existingNotifiers)

	assert.Emptyf(t, existingButNotImported, "some existing notifiers aren't imported")
	assert.Empty(t, importedButNotExisting, "some imported notifiers don't exist")
}
