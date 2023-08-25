package operations

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
)

func renderCountFunctionSignature(statement *Statement) *Statement {
	functionName := "Count"
	return statement.Id(functionName).Params(
		Id("ctx").Qual("context", "Context"),
		Id("q").Op("*").Qual(packagenames.V1, "Query"),
	).Parens(List(Int(), Error()))
}

func generateCount(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderCountFunctionSignature(&Statement{})

	mappingPath := GenerateMappingGoPackage(props)
	implementation := renderCountFunctionSignature(renderFuncBStarIndexer()).Block(
		metricLine("Count", props.Object),
		Return(Qual(packagenames.RoxBleve, "RunCountRequest").Call(Qual(packagenames.V1, props.SearchCategory), Id("q"), Id("b").Dot("index"), Qual(mappingPath, "OptionsMap"), Id("opts").Op("..."))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["count"] = generateCount
}
