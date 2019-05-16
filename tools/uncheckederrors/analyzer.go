package uncheckederrors

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `check for unchecked errors returned from funcs`

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "uncheckederrors",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var (
	errType = types.Universe.Lookup("error").Type()

	// whitelist contains functions from the standard library where we're okay with not
	// checking returned errors.
	whitelist = map[string]set.FrozenStringSet{
		"fmt": set.NewFrozenStringSet(
			"Println",
			"Printf",
			"Fprint",
			"Fprintf",
			"Fprintln",
		),
	}
)

// unparen returns e with any enclosing parentheses stripped.
func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

func getReturnTypesFromFuncType(typ types.Type) (typs []types.Type) {
	for typ.Underlying() != typ {
		typ = typ.Underlying()
	}
	sig, ok := typ.(*types.Signature)
	if !ok {
		panic("type was not a signature")
	}
	results := sig.Results()
	if results == nil {
		return
	}
	for i := 0; i < results.Len(); i++ {
		typs = append(typs, results.At(i).Type())
	}
	return
}

func getReturnTypesFromFuncObj(obj types.Object) (objName string, typs []types.Type) {
	objName = obj.Name()

	typs = getReturnTypesFromFuncType(obj.Type())
	return
}

func getReturnTypesOfFunc(pass *analysis.Pass, fun ast.Expr) (name string, typs []types.Type, exprs []ast.Expr) {
	switch fun := fun.(type) {
	case *ast.Ident:
		name, typs = getReturnTypesFromFuncObj(pass.TypesInfo.Uses[fun])
		return
	case *ast.SelectorExpr:
		if obj, ok := pass.TypesInfo.Selections[fun]; ok {
			name, typs = getReturnTypesFromFuncObj(obj.Obj())
			return
		}
		if obj, ok := pass.TypesInfo.Uses[fun.Sel]; ok {
			name, typs = getReturnTypesFromFuncObj(obj)
			return

		}
	case *ast.FuncLit:
		name = "inline_func"
		if fun.Type != nil && fun.Type.Results != nil {
			for _, result := range fun.Type.Results.List {
				exprs = append(exprs, result.Type)
			}
		}
		return
	// This happens if you call a function which is returned by another function.
	// Example: you have a func a() func(string) error,
	// and you call a()("123"), we see a() as a call expression.
	// In this case, we assert that the func has exactly one return value, which is a function,
	// and check whether the returned function returns an error by recursively calling this function.
	case *ast.CallExpr:
		_, typs, exprs := getReturnTypesOfFunc(pass, fun.Fun)
		if len(typs)+len(exprs) != 1 {
			panic(fmt.Sprintf("Got func %+v that doesn't return exactly one element", fun.Fun))
		}
		name = "function_that_returns_a_func"
		if len(typs) == 1 {
			return name, getReturnTypesFromFuncType(typs[0]), nil
		}
		if len(exprs) == 1 {
			_, typs, exprs := getReturnTypesOfFunc(pass, exprs[0])
			return name, typs, exprs
		}
	}

	panic(fmt.Sprintf("Unexpected func type %T", fun))
}

func doesFuncReturnError(pass *analysis.Pass, fun ast.Expr) (name string, returnsError bool) {
	name, typs, exprs := getReturnTypesOfFunc(pass, fun)
	for _, typ := range typs {
		if types.Identical(typ, errType) {
			returnsError = true
		}
	}
	for _, expr := range exprs {
		if expr, ok := expr.(*ast.Ident); ok {
			if expr.Name == "error" {
				returnsError = true
			}
		}
	}
	return
}

func isWhitelisted(fun ast.Expr) bool {
	if fun, ok := fun.(*ast.SelectorExpr); ok {
		if pkg, ok := fun.X.(*ast.Ident); ok {
			if whitelistSet, ok := whitelist[pkg.Name]; ok {
				if whitelistSet.Contains(fun.Sel.Name) {
					return true
				}
			}
		}
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ExprStmt)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.GoStmt)(nil),
	}
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		var potentialCallExpr ast.Expr
		switch n := n.(type) {
		case *ast.ExprStmt:
			potentialCallExpr = n.X
		case *ast.DeferStmt:
			potentialCallExpr = n.Call
		case *ast.GoStmt:
			potentialCallExpr = n.Call
		default:
			panic(fmt.Sprintf("Unexpected type: %T", n))
		}
		call, ok := unparen(potentialCallExpr).(*ast.CallExpr)
		if !ok {
			return // not a call statement
		}
		fun := unparen(call.Fun)

		if pass.TypesInfo.Types[fun].IsType() || pass.TypesInfo.Types[fun].IsBuiltin() {
			return // a type conversion, or a builtin (like panic)
		}

		if isWhitelisted(fun) {
			return
		}
		objName, returnsError := doesFuncReturnError(pass, fun)
		if returnsError {
			pass.Reportf(call.Lparen, "error result of func %s is not being checked", objName)
		}
	})
	return nil, nil
}
