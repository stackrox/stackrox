package testtags

import (
	"go/ast"
	"path/filepath"
	"strings"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `Ensure that our *_test.go files include a //go:build <TAG>, otherwise they won't be run by the test target.
You can run scripts/prepend-go-build-to-tests.sh <BUILD_TAG> with the tag corresponding to your test target, to add it
 to all *_test.go files in the directory.'`

const roxPrefix = "github.com/stackrox/rox/"

// Analyzer is a analysis.Analyzer from the analysis package of the Go standard lib. [It analyzes code]
var Analyzer = &analysis.Analyzer{
	Name:     "testtags",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Make sure this runs only on tests within stackrox/tests, our go e2e tests
func isTestsPackage(packageName string) bool {
	if !strings.HasPrefix(packageName, roxPrefix) {
		return false
	}
	unqualifiedPackageName := strings.TrimPrefix(packageName, roxPrefix)
	pathElems := strings.Split(unqualifiedPackageName, string(filepath.Separator))
	if len(pathElems) == 0 {
		return false
	}
	if pathElems[0] == "tests" {
		return true
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
	}

	if !isTestsPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	common.FilteredPreorder(inspectResult, common.IsTestFile, nodeFilter, func(n ast.Node) {
		hasGoBuildDirective := false
		fileNode := n.(*ast.File)
		pos := fileNode.Pos()
		fileContainsTestFunction := false
		for _, decl := range fileNode.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if strings.HasPrefix(d.Name.String(), "Test") {
					fileContainsTestFunction = true
					break
				}
			}
		}
		if fileContainsTestFunction {
			for _, comment := range fileNode.Comments {
				if strings.HasPrefix(comment.Text(), "//go:build") {
					hasGoBuildDirective = true
				}
			}
			if !hasGoBuildDirective {
				pass.Reportf(pos, "Missing //go:build directive.")
			}
		}
	})
	return nil, nil
}
