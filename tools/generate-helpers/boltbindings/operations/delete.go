package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
)

func renderDeleteFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Delete%s", props.Singular)
	return statement.Id(functionName).Params(Id("id").String()).Error()
}

func generateDelete(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderDeleteFunctionSignature(&Statement{}, props)

	implementation := renderDeleteFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("Remove", props.Singular),
		List(Id("_"), Id("_"), Err()).Op(":=").Id("s").Dot("crud").Dot("Delete").Call(Id("id")),
		Return(Err()),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["delete"] = generateDelete
}
