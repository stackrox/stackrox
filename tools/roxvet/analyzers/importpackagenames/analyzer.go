package importpackagenames

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = `check that we import certain packages using particular names for consistency`

var rules = []struct {
	description      string
	importPathSuffix string
	expectedName     string
}{
	{
		"controller-runtime client",
		"controller-runtime/pkg/client",
		"ctrlClient",
	},
}

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "importpackagenames",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			verifyImportUsesAllowedPackageName(pass, imp)
		}
	}

	return nil, nil
}

// verifyImportUsesAllowedPackageName verifies that if this imports a path for which
// there is a rule, it uses the appropriate package name.
func verifyImportUsesAllowedPackageName(pass *analysis.Pass, imp *ast.ImportSpec) {
	for _, rule := range rules {
		if !strings.HasSuffix(imp.Path.Value, rule.importPathSuffix+"\"") {
			continue
		}
		if imp.Name == nil || imp.Name.Name != rule.expectedName {
			pass.Reportf(imp.Pos(), "inconsistent package name for import %s: please import %s as %q", imp.Path.Value, rule.description, rule.expectedName)
		}
	}
}
