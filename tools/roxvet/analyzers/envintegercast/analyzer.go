package envintegercast

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const envPkgPath = "github.com/stackrox/rox/pkg/env"

const doc = `check for direct narrowing casts from env integer settings`

var narrowedIntegerCasts = map[string]struct{}{
	"int8":   {},
	"int16":  {},
	"int32":  {},
	"uint8":  {},
	"uint16": {},
	"uint32": {},
}

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "envintegercast",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		cast, ok := call.Fun.(*ast.Ident)
		if !ok {
			return
		}
		if _, ok := narrowedIntegerCasts[cast.Name]; !ok || len(call.Args) != 1 {
			return
		}
		if !isIntegerSettingCall(pass, call.Args[0]) {
			return
		}
		pass.Reportf(call.Pos(), "avoid direct %s conversion of IntegerSetting(); validate or bound the setting before narrowing", cast.Name)
	})
	return nil, nil
}

func isIntegerSettingCall(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "IntegerSetting" || len(call.Args) != 0 {
		return false
	}

	selection := pass.TypesInfo.Selections[selector]
	if selection == nil {
		return false
	}
	return isEnvIntegerSettingMethod(selection.Obj())
}

func isEnvIntegerSettingMethod(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != envPkgPath {
		return false
	}
	return fn.Name() == "IntegerSetting"
}
