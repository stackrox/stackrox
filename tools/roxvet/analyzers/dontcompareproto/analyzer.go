package donotcompareproto

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

const doc = `Inspect calls to Equal for proto arguments that should be compared with protocompat.Equal instead`

// Analyzer is the go vet entrypoint
var Analyzer = &analysis.Analyzer{
	Name:     "donotcompareproto",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var bannedFunctions = set.NewFrozenStringSet(
	//"(*github.com/stretchr/testify/assert.Assertions).Contains",
	//"(*github.com/stretchr/testify/assert.Assertions).Equal",
	//"(*github.com/stretchr/testify/assert.Assertions).NotEqual",
	//"(*github.com/stretchr/testify/require.Assertions).Contains",
	//"(*github.com/stretchr/testify/require.Assertions).Equal",
	//"(*github.com/stretchr/testify/require.Assertions).NotEqual",
	////TODO(janisz): enable it in a separated PR
	////"github.com/google/go-cmp/cmp.Diff",
	////"github.com/google/go-cmp/cmp.Equal",
	//"github.com/stretchr/testify/assert.Contains",
	//"github.com/stretchr/testify/assert.Equal",
	//"github.com/stretchr/testify/assert.NotEqual",
	//"github.com/stretchr/testify/require.Contains",
	//"github.com/stretchr/testify/require.Equal",
	//"github.com/stretchr/testify/require.NotEqual",
	"reflect.DeepEqual",
)

func run(pass *analysis.Pass) (interface{}, error) {
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
		has, expr := hasStorageArg(call, pass)
		if !has {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:     expr.Pos(),
			Message: "Use protocompat.Equal to compare proto.Message",
		})
	})
	return nil, nil
}

func hasStorageArg(call *ast.CallExpr, pass *analysis.Pass) (bool, ast.Expr) {
	for _, arg := range call.Args {
		if isStorage(pass.TypesInfo, arg) {
			return true, arg
		}
	}
	return false, ast.Expr(nil)
}

func isStorage(info *types.Info, expr ast.Expr) bool {
	typ := info.Types[expr].Type
	return typ != nil &&
		strings.Contains(typ.String(), "*github.com/stackrox/rox/generated/storage.") &&
		typ.Underlying().String() != "int32"
}
