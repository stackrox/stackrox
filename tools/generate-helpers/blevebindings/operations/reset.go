package operations

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/blevebindings/packagenames"
)

func renderResetFunctionSignature(statement *Statement, props GeneratorProperties) *Statement {
	return statement.Id("ResetIndex").Params().Error()
}

func generateReset(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderResetFunctionSignature(&Statement{}, props)

	implementation := renderResetFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("Reset", props.Object),
		Return(Qual(packagenames.RoxBleve, "ResetIndex").Call(Qual(packagenames.V1, props.SearchCategory), Id("b").Dot("index").Dot("Index"))),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["reset"] = generateReset
}
