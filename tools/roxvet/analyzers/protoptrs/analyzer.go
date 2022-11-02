package protoptrs

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	protoMsgType = types.NewInterfaceType(
		[]*types.Func{
			types.NewFunc(token.NoPos, nil, "ProtoMessage", types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)),
		},
		nil).Complete()

	protoTypesRegex = regexp.MustCompile(`^github\.com/stackrox/rox/generated[./]|^github\.com/gogo/protobuf/types[./]`)
)

// Analyzer is the go vet entrypoint
var Analyzer = &analysis.Analyzer{
	Name:     "protoptrs",
	Doc:      "checks that protobuf message types are only used as pointers",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncType)(nil),
		(*ast.StructType)(nil),
	}
	common.FilteredPreorder(inspectResult, common.Not(common.IsGeneratedFile), nodeFilter, func(n ast.Node) {
		var relevantFields []*ast.Field
		switch t := n.(type) {
		case *ast.FuncType:
			if t.Params != nil {
				relevantFields = append(relevantFields, t.Params.List...)
			}
			if t.Results != nil {
				relevantFields = append(relevantFields, t.Results.List...)
			}
		case *ast.StructType:
			if t.Fields != nil {
				relevantFields = append(relevantFields, t.Fields.List...)
			}
		}

		for _, field := range relevantFields {
			ty := pass.TypesInfo.TypeOf(field.Type)
			if ty == nil {
				continue
			}
			if named, _ := ty.(*types.Named); named == nil || !protoTypesRegex.MatchString(named.String()) {
				continue
			}
			structTy, _ := ty.Underlying().(*types.Struct)
			if structTy == nil {
				continue
			}
			tyPtr := types.NewPointer(ty)
			if !types.Implements(tyPtr, protoMsgType) {
				continue
			}
			pass.Report(analysis.Diagnostic{
				Pos:     field.Type.Pos(),
				Message: "Always use pointers to proto message types in parameters, return values, and struct fields",
			})
		}
	})
	return nil, nil
}
