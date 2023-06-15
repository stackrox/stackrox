package operations

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
)

func renderSearchFunctionSignature(statement *Statement) *Statement {
	functionName := "Search"
	return statement.Id(functionName).Params(
		Id("ctx").Qual("context", "Context"),
		Id("q").Op("*").Qual(packagenames.V1, "Query"),
	).Parens(List(Index().Qual(packagenames.RoxSearch, "Result"), Error()))
}

func generateSearch(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderSearchFunctionSignature(&Statement{})

	mappingPath := GenerateMappingGoPackage(props)
	implementation := renderSearchFunctionSignature(renderFuncBStarIndexer()).Block(
		metricLine("Search", props.Object),
		Return(Qual(packagenames.RoxBleve, "RunSearchRequest").Call(Qual(packagenames.V1, props.SearchCategory), Id("q"), Id("b").Dot("index"), Qual(mappingPath, "OptionsMap"), Id("opts").Op("..."))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["search"] = generateSearch
}
