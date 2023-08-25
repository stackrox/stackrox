package needlessformat

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "needlessformat",
	Doc:      "check for use of format methodsToReplacementByPackage without format arguments",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var (
	methodsToReplacementByPackage = map[string]map[string]string{
		"fmt": {
			"Printf":  "Print",
			"Fprintf": "Fprint",
			"Sprintf": "Sprint or remove",
			"Errorf":  "errors.New",
		},
		"github.com/stackrox/rox/pkg/errorhelpers": {
			"(*ErrorList).AddWrapf":   "AddWrap",
			"(*ErrorList).AddStringf": "AddString",
		},
		"github.com/stackrox/rox/pkg/logging": {
			"(*Logger).Infof":  "Info",
			"(*Logger).Warnf":  "Warn",
			"(*Logger).Debugf": "Debug",
			"(*Logger).Errorf": "Error",
			"(*Logger).Panicf": "Panic",
			"(*Logger).Fatalf": "Fatal",
		},
		"github.com/pkg/errors": {
			"Errorf": "New",
			"Wrapf":  "Wrap",
		},
		"google.golang.org/grpc/status": {
			"Errorf": "Error",
		},
	}
)

func isNeedlessVarArgsCall(fun *types.Func, call *ast.CallExpr) (bool, string, string) {
	sig := fun.Type().(*types.Signature)
	if sig == nil || !sig.Variadic() || len(call.Args) != sig.Params().Len()-1 {
		return false, "", ""
	}

	methodsToReplacement := methodsToReplacementByPackage[fun.Pkg().Path()]
	if methodsToReplacement == nil {
		return false, "", ""
	}

	name := fun.Name()
	if sig.Recv() != nil {
		recvTy := sig.Recv().Type()
		qf := types.RelativeTo(fun.Pkg())
		if ptrTy, _ := recvTy.(*types.Pointer); ptrTy != nil {
			name = fmt.Sprintf("(*%s).%s", types.TypeString(ptrTy.Elem(), qf), name)
		} else {
			name = fmt.Sprintf("%s.%s", types.TypeString(recvTy, qf), name)
		}
	}
	replacement, match := methodsToReplacement[name]
	return match, fmt.Sprintf("\"%s\".%s", fun.Pkg().Path(), name), replacement
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	fun := astutil.Unparen(call.Fun)

	if pass.TypesInfo.Types[fun].IsType() || pass.TypesInfo.Types[fun].IsBuiltin() {
		return // a type conversion, or a builtin (like panic)
	}

	namedFun, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if namedFun == nil {
		return
	}

	if match, name, replacement := isNeedlessVarArgsCall(namedFun, call); match {
		pass.Reportf(call.Fun.Pos(), "Varargs function %s called with no variadic arguments; use %s instead", name, replacement)
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	common.FilteredPreorder(inspectResult, common.Not(common.IsGeneratedFile), nodeFilter, func(n ast.Node) {
		checkCall(pass, n.(*ast.CallExpr))
	})
	return nil, nil
}
