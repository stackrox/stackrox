package testtags

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = `Ensure that our *_test.go files include a //go:build <TAG>, otherwise they won't be run.'`

// Analyzer is a analysis.Analyzer from the analysis package of the Go standard lib. [It analyzes code]
var Analyzer = &analysis.Analyzer{
	Name:     "testtags",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	packagePath := pass.Pkg.Path()
	goTestFiles, err := filepath.Glob(filepath.Join(packagePath, "*_test.go"))
	if err != nil {
		pass.Reportf(token.NoPos, "Failed to parse package for files: %v", err)
		return nil, nil
	}

	for _, filePath := range goTestFiles {
		var fileContent []byte
		fileContent, err = os.ReadFile(filePath)
		if err != nil {
			pass.Reportf(token.NoPos, "os.ReadFile failed with: %v", err)
			return nil, nil
		}
		numGoBuildDirectives := numGoBuildDirectives(string(fileContent))
		if numGoBuildDirectives == 0 {
			pass.Reportf(token.NoPos, "\"%s\" is missing a //go:build directive.", filePath)
		} else if numGoBuildDirectives > 1 {
			pass.Reportf(token.NoPos, "\"%s\" has multiple //go:build directives, there should only be one.", filePath)
		}
	}
	return nil, nil
}

func numGoBuildDirectives(fileContent string) int {
	lines := strings.Split(fileContent, "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "//go:build") {
			count += 1
		}
	}
	return count
}
