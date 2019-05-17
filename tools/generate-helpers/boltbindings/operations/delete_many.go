package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderDeleteManyFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Delete%s", props.Plural)
	return statement.Id(functionName).Params(jen.Id("ids").Index().String()).Error()
}

func generateDeleteMany(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderDeleteManyFunctionSignature(&jen.Statement{}, props)

	implementation := renderDeleteManyFunctionSignature(renderFuncSStarStore(), props).Block(
		jen.Return(jen.Id("s").Dot("crud").Dot("DeleteBatch").Call(jen.Id("ids"))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["delete_many"] = generateDeleteMany
}
