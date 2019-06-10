package operations

import (
	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/blevebindings/packagenames"
)

func renderResetFunctionSignature(statement *jen.Statement, props GeneratorProperties) *jen.Statement {
	return statement.Id("ResetIndex").Params().Error()
}

func generateReset(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderResetFunctionSignature(&jen.Statement{}, props)

	implementation := renderResetFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("Reset", props.Object),
		jen.Return(jen.Qual(packagenames.RoxBleve, "ResetIndex").Call(jen.Qual(packagenames.V1, props.SearchCategory), jen.Id("b").Dot("index").Dot("Index"))),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["reset"] = generateReset
}
