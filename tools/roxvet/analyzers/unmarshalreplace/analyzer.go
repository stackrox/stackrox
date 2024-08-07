package unmarshalreplace

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `Direct calls to jsonpb.Unmarshal should be replaced with jsonutil.JSONReaderToProto or jsonutil.JSONBytesToProto`

var Analyzer = &analysis.Analyzer{
	Name:     "unmarshalreplace",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var allowedCallerPackages = []string{}

var bannedFunctions = set.NewFrozenStringSet(
	"github.com/golang/protobuf/jsonpb.Unmarshal",
)

func run(pass *analysis.Pass) (interface{}, error) {
	callerPkg := pass.Pkg.Path()
	for _, allowedPkg := range allowedCallerPackages {
		if strings.HasPrefix(callerPkg, allowedPkg) {
			return nil, nil
		}
	}
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if !ok || !bannedFunctions.Contains(fn.FullName()) {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			Message: "Use jsonutil.JSONReaderToProto instead of jsonpb.Unmarshal",
		})
	})
	return nil, nil
}
