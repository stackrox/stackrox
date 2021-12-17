package filepathwalk

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	doc = `check for usages of filepath.Walk`

	filepathWalk = `path/filepath.Walk`
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "filepathwalk",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if ok && fn.FullName() == filepathWalk {
			pass.Report(analysis.Diagnostic{
				Pos:     n.Pos(),
				Message: "Use filepath.WalkDir instead, as it is more efficient (https://pkg.go.dev/path/filepath#WalkDir).",
			})
		}
	})
	return nil, nil
}
