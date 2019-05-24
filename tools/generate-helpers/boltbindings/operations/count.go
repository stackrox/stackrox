package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderCountFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Count%s", props.Plural)
	return statement.Id(functionName).Params().Parens(jen.List(jen.Id("count").Int(), jen.Err().Error()))
}

func generateCount(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderCountFunctionSignature(&jen.Statement{}, props)

	implementation := renderCountFunctionSignature(renderFuncSStarStore(), props).Block(
		metricLine("Count", props.Singular),
		jen.Return(jen.Id("s").Dot("crud").Dot("Count").Call()),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["count"] = generateCount
}
