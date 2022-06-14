package runner

import (
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
		if !strings.HasPrefix(baseName, "m_") {
			continue
		}
		existingMigrations.Add(baseName)
	}

	f, err := parser.ParseFile(token.NewFileSet(), "all.go", nil, parser.ImportsOnly)
	require.NoError(t, err, "failed to parse all.go")

	importedMigrations := set.NewStringSet()
	for _, imp := range f.Imports {
		pkgName := strings.TrimSuffix(strings.TrimPrefix(imp.Path.Value, `"`), `"`)
		pkgBaseName := path.Base(pkgName)
		if !strings.HasPrefix(pkgBaseName, "m_") {
			continue
		}
		importedMigrations.Add(pkgBaseName)
	}

	existingButNotImported := existingMigrations.Difference(importedMigrations)
	importedButNotExisting := importedMigrations.Difference(existingMigrations)

	assert.Empty(t, existingButNotImported, "some existing migrations aren't imported")
	assert.Empty(t, importedButNotExisting, "some imported migrations don't exist")
}
