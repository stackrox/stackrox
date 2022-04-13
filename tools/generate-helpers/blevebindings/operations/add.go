package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/stackrox/tools/generate-helpers/common/packagenames"
)

func renderAddFunctionSignature(statement *Statement, props GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Add%s", props.Singular)
	return statement.Id(functionName).Params(Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateAdd(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderAddFunctionSignature(&Statement{}, props)

	wrapperType := MakeWrapperType(props.Object)

	implementation := renderAddFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("Add", props.Object),
		ifErrReturnError(Id("b").Dot("index").Dot("Index").Call(
			Id(strings.ToLower(props.Singular)).Dot(props.IDFunc).Call(),
			Op("&").Id(wrapperType).Values(Dict{
				Id("Type"):       Qual(packagenames.V1, props.SearchCategory).Dot("String").Call(),
				Id(props.Object): Id(strings.ToLower(props.Singular)),
			}),
		)),
		Return(Nil()),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add"] = generateAdd
}
