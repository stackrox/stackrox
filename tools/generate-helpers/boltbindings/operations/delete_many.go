package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
)

func renderDeleteManyFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Delete%s", props.Plural)
	return statement.Id(functionName).Params(Id("ids").Index().String()).Error()
}

func generateDeleteMany(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderDeleteManyFunctionSignature(&Statement{}, props)

	implementation := renderDeleteManyFunctionSignature(renderFuncSStarStore(), props).Block(
		metricLine("RemoveMany", props.Singular),
		Return(Id("s").Dot("crud").Dot("DeleteBatch").Call(Id("ids"))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["delete_many"] = generateDeleteMany
}
