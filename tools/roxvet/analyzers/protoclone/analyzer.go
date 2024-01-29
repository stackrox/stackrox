package protoclone

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	doc = `check for usages of proto.Clone followed by type assertions`

	gogoProtoPkg = "github.com/gogo/protobuf/proto"
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "protoclone",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	nodeFilter := []ast.Node{(*ast.TypeAssertExpr)(nil)}

	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		typeAssertNode := n.(*ast.TypeAssertExpr)
		callExpr, _ := typeAssertNode.X.(*ast.CallExpr)
		if callExpr == nil {
			return false
		}
		calledFunc := typeutil.Callee(pass.TypesInfo, callExpr)
		if calledFunc == nil || calledFunc.Pkg() == nil {
			return false
		}
		if calledFunc.Pkg().Path() == gogoProtoPkg && calledFunc.Name() == "Clone" {
			pass.Reportf(n.Pos(), "do not call proto.Clone and cast, just use .Clone() on the object directly")
		}
		return false
	})

	return nil, nil
}
