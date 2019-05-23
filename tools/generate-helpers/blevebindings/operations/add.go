package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/blevebindings/packagenames"
)

func renderAddFunctionSignature(statement *jen.Statement, props GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Add%s", props.Singular)
	return statement.Id(functionName).Params(jen.Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateAdd(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderAddFunctionSignature(&jen.Statement{}, props)

	wrapperType := MakeWrapperType(props.Object)

	implementation := renderAddFunctionSignature(renderFuncBStarIndexer(), props).Block(
		jen.Return(jen.Id("b").Dot("index").Dot("Index").Call(
			jen.Id(strings.ToLower(props.Singular)).Dot(props.IDFunc).Call(),
			jen.Op("&").Id(wrapperType).Values(jen.Dict{
				jen.Id("Type"):       jen.Qual(packagenames.V1, props.SearchCategory).Dot("String").Call(),
				jen.Id(props.Object): jen.Id(strings.ToLower(props.Singular)),
			}),
		)),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add"] = generateAdd
}
