package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
)

func renderGetFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Get%s", props.HumanName)
	return statement.Id(functionName).Params().Parens(List(
		Op("*").Qual(props.Pkg, props.Object), Error(),
	))
}

// GenerateGet generates the get method.
func GenerateGet(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderGetFunctionSignature(&Statement{}, props)

	implementation := renderGetFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("Get", props.HumanName),
		List(Id("msg"), Err()).Op(":=").Id("s").Dot("underlying").Dot("Get").Call(),
		If(Err().Op("!=").Nil()).Block(
			Return(Nil(), Err()),
		),
		If(Id("msg").Op("==").Nil()).Block(
			Return(Nil(), Nil()),
		),
		Return(Id("msg").Assert(Op("*").Qual(props.Pkg, props.Object)), Nil()),
	)

	return interfaceMethod, implementation
}
