package sortslices

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `check for usages of "sort" funcs which should be replaced with equivalent "slices" funcs`

// slicesFuncs maps the sort functions to the slices equivalent.
//
// Note: sort.IsSorted, sort.Sort, and sort.Stable are not considered, as they take in a sort.Interface which
// may not necessarily be a slice. Perhaps the usage of these functions should be audited and considered for
// replacement. The functions sort.Slice and sort.SliceStable should also be considered.
var slicesFuncs = map[string]string{
	`sort.Float64s`:          `slices.Sort`,
	`sort.Float64sAreSorted`: `slices.IsSorted`,
	`sort.Ints`:              `slices.Sort`,
	`sort.IntsAreSorted`:     `slices.IsSorted`,
	`sort.Strings`:           `slices.Sort`,
	`sort.StringsAreSorted`:  `slices.IsSorted`,
}

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "sortslices",
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
		if !ok {
			return
		}
		slicesFunc, ok := slicesFuncs[fn.FullName()]
		if !ok {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:     n.Pos(),
			Message: fmt.Sprintf("Use %s instead of %s, as it is more efficient/ergonomic (https://pkg.go.dev/slices).", slicesFunc, fn.FullName()),
		})
	})
	return nil, nil
}
