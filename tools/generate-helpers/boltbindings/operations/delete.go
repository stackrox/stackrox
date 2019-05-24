package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
)

func renderDeleteFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Delete%s", props.Singular)
	return statement.Id(functionName).Params(Id("id").String()).Error()
}

func generateDelete(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderDeleteFunctionSignature(&Statement{}, props)

	implementation := renderDeleteFunctionSignature(renderFuncSStarStore(), props).Block(
		metricLine("Remove", props.Singular),
		Return(Id("s").Dot("crud").Dot("Delete").Call(Id("id"))),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["delete"] = generateDelete
}
