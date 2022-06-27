package uncheckederrors

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
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

	// allow list contains functions from the standard library where we're okay with not
	// checking returned errors.
	allowList = map[string]set.FrozenStringSet{
		"bytes": set.NewFrozenStringSet(
			"(*Buffer).WriteString",
		),
		"fmt": set.NewFrozenStringSet(
			"Print",
			"Println",
			"Printf",
			"Fprint",
			"Fprintf",
			"Fprintln",
		),
		"strings": set.NewFrozenStringSet(
			"(*Builder).WriteString",
			"(*Builder).WriteRune",
			"(*Builder).WriteByte",
		),
		"github.com/stackrox/rox/pkg/utils": set.NewFrozenStringSet(
			"Should",
		),
	}
)

func getReturnTypesFromFuncType(typ types.Type) (typs []types.Type) {
	for {
		// This deferences type definitions.
		// For example, if you have:
		// type myFunc func() error
		// Then, if typ is myFunc, then
		// typ.Underlying() is func() error
		for typ.Underlying() != typ {
			typ = typ.Underlying()
		}
		// This dereferences pointers, arrays and slices.
		elemTyp, canElem := typ.(interface{ Elem() types.Type })
		if !canElem {
			break
		}
		typ = elemTyp.Elem()
	}
	sig, ok := typ.(*types.Signature)
	if !ok {
		panic(fmt.Sprintf("type %+v was not a signature", typ))
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
	// This handles the case where you have
	// var funcs []func() error
	// funcs[2]() <- unchecked error.
	case *ast.IndexExpr:
		return getReturnTypesOfFunc(pass, fun.X)
	}

	panic(fmt.Sprintf("Unexpected func type %T %+v", fun, fun))
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

func isAllowlisted(fun *types.Func) bool {
	if allowListSet, ok := allowList[fun.Pkg().Path()]; ok {
		name := fun.Name()
		sig := fun.Type().(*types.Signature)
		if sig != nil && sig.Recv() != nil {
			recvTy := sig.Recv().Type()
			qf := types.RelativeTo(fun.Pkg())
			if ptrTy, _ := recvTy.(*types.Pointer); ptrTy != nil {
				name = fmt.Sprintf("(*%s).%s", types.TypeString(ptrTy.Elem(), qf), name)
			} else {
				name = fmt.Sprintf("%s.%s", types.TypeString(recvTy, qf), name)
			}
		}
		if allowListSet.Contains(name) {
			return true
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
		call, ok := astutil.Unparen(potentialCallExpr).(*ast.CallExpr)
		if !ok {
			return // not a call statement
		}
		fun := astutil.Unparen(call.Fun)

		if pass.TypesInfo.Types[fun].IsType() || pass.TypesInfo.Types[fun].IsBuiltin() {
			return // a type conversion, or a builtin (like panic)
		}

		namedFun, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if namedFun != nil && isAllowlisted(namedFun) {
			return
		}
		objName, returnsError := doesFuncReturnError(pass, fun)
		if returnsError {
			pass.Reportf(call.Lparen, "error result of func %s is not being checked", objName)
		}
	})
	return nil, nil
}
