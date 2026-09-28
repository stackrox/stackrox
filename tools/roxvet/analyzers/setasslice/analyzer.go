package setasslice

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `check for .AsSlice() calls in range loops over set types`

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "setasslice",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const setPkgPath = "github.com/stackrox/rox/pkg/set"

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.RangeStmt)(nil)}
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		rangeStmt := n.(*ast.RangeStmt)
		call, ok := rangeStmt.X.(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "AsSlice" {
			return
		}
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if !ok {
			return
		}
		sig, ok := fn.Type().(*types.Signature)
		if !ok || sig.Recv() == nil {
			return
		}
		recvType := sig.Recv().Type().String()
		if !strings.HasPrefix(recvType, setPkgPath+".") {
			return
		}
		if strings.Contains(recvType, "FrozenSet") {
			pass.Reportf(sel.Sel.Pos(), "use .All() instead of .AsSlice() in range loops to avoid allocation")
		} else {
			pass.Reportf(sel.Sel.Pos(), "range over the set directly instead of calling .AsSlice()")
		}
	})
	return nil, nil
}
