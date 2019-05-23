package operations

import (
	"path"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/blevebindings/packagenames"
)

func renderSearchFunctionSignature(statement *jen.Statement) *jen.Statement {
	functionName := "Search"
	return statement.Id(functionName).Params(jen.Id("q").Op("*").Qual(packagenames.V1, "Query")).Parens(jen.List(jen.Index().Qual(packagenames.RoxSearch, "Result"), jen.Error()))
}

func generateSearch(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderSearchFunctionSignature(&jen.Statement{})

	mappingPath := path.Join(packagenames.RoxCentral, strings.ToLower(props.Object), packagenames.RoxMappingSubPath)
	implementation := renderSearchFunctionSignature(renderFuncBStarIndexer()).Block(
		jen.Return(jen.Qual(packagenames.RoxBleve, "RunSearchRequest").Call(jen.Qual(packagenames.V1, props.SearchCategory), jen.Id("q"), jen.Id("b").Dot("index"), jen.Qual(mappingPath, "OptionsMap"))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["search"] = generateSearch
}
