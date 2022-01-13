package dontprintferr

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `Inspect fmt.Errorf calls for error arguments that should be wrapped with errors.Wrap() instead`

// Analyzer is the go vet entrypoint
var Analyzer = &analysis.Analyzer{
	Name:     "dontprintferr",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if ok && fn.FullName() == "fmt.Errorf" {
			// Check if the format string contains %w (new style error wrapping, allowed)
			lit := pass.TypesInfo.Types[call.Args[0]].Value
			if lit != nil && lit.Kind() == constant.String && strings.Contains(constant.StringVal(lit), "%w") {
				return
			}
			for _, arg := range call.Args[1:] {
				if matchType(pass.TypesInfo, arg, "error") {
					pass.Report(analysis.Diagnostic{
						Pos:     arg.Pos(),
						Message: "Use '%w' when wrapping errors using fmt.Errorf, or use pkg/errors.(Wrap/Wrapf)",
					})
				}
			}
		}
	})
	return nil, nil
}

func matchType(info *types.Info, expr ast.Expr, want string) bool {
	typ := info.Types[expr].Type
	return typ != nil && typ.String() == want
}
