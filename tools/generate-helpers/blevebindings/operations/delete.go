package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderDeleteFunctionSignature(statement *jen.Statement, props GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Delete%s", props.Singular)
	return statement.Id(functionName).Params(jen.Id("id").String()).Error()
}

func generateDelete(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderDeleteFunctionSignature(&jen.Statement{}, props)

	implementation := renderDeleteFunctionSignature(renderFuncBStarIndexer(), props).Block(
		jen.Return(jen.Id("b").Dot("index").Dot("Delete").Call(jen.Id("id"))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["delete"] = generateDelete
}
