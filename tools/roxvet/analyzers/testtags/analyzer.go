package testtags

import (
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = `Ensure that our *_test.go files include a //go:build <TAG>, otherwise they won't be run.'`

const roxPrefix = "github.com/stackrox/rox/"

// Analyzer is a analysis.Analyzer from the analysis package of the Go standard lib. [It analyzes code]
var Analyzer = &analysis.Analyzer{
	Name:     "testtags",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Given the package name, get the root directory of the service.
func isTest(packageName string) bool {
	if !strings.HasPrefix(packageName, roxPrefix) {
		return false
	}
	unqualifiedPackageName := strings.TrimPrefix(packageName, roxPrefix)
	pathElems := strings.Split(unqualifiedPackageName, string(filepath.Separator))
	if pathElems[0] == "tests" {
		return true
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	packagePath := pass.Pkg.Path()
	if !isTest(packagePath) {
		return nil, nil
	}
	root := strings.TrimPrefix(packagePath, roxPrefix)
	root = "../../../../../" + root

	var goTestFiles []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(path, "_test.go") {
			goTestFiles = append(goTestFiles, path)
		}
		return nil
	})

	if err != nil {
		pass.Reportf(token.NoPos, "Failed to walk directory %s with error %v", packagePath, err)
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
