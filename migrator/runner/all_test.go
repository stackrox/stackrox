package runner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllPackagesAreImported(t *testing.T) {
	migrationDirEntries, err := os.ReadDir("../migrations")
	require.NoError(t, err, "failed to read migrations directory")

	existingMigrations := set.NewStringSet()
	for _, entry := range migrationDirEntries {
		if !entry.IsDir() {
			continue
		}
		baseName := filepath.Base(entry.Name())

		if !isMigrationName(baseName) {
			continue
		}
		existingMigrations.Add(baseName)
	}

	var allImports []*ast.ImportSpec
	f, err := parser.ParseFile(token.NewFileSet(), "all.go", nil, parser.ImportsOnly)
	require.NoError(t, err, "failed to parse all.go")

	allImports = append(allImports, f.Imports...)

	f, err = parser.ParseFile(token.NewFileSet(), "all_rocksdb.go", nil, parser.ImportsOnly)
	require.NoError(t, err, "failed to parse all_rocksdb.go")

	allImports = append(allImports, f.Imports...)

	importedMigrations := set.NewStringSet()
	for _, imp := range allImports {
		pkgName := strings.TrimSuffix(strings.TrimPrefix(imp.Path.Value, `"`), `"`)
		pkgBaseName := path.Base(pkgName)
		if !isMigrationName(pkgBaseName) {
			continue
		}
		importedMigrations.Add(pkgBaseName)
	}

	existingButNotImported := existingMigrations.Difference(importedMigrations)
	importedButNotExisting := importedMigrations.Difference(existingMigrations)

	assert.Empty(t, existingButNotImported, "some existing migrations aren't imported")
	assert.Empty(t, importedButNotExisting, "some imported migrations don't exist")
}

func isMigrationName(name string) bool {
	return strings.HasPrefix(name, "m_") || strings.HasPrefix(name, "n_")
}
