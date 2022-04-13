package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/stackrox/tools/generate-helpers/common"
)

func renderGetFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Get%s", props.Singular)
	returns := []Code{Op("*").Qual(props.Pkg, props.Object)}
	if props.GetExists {
		returns = append(returns, Bool())
	}
	returns = append(returns, Error())
	return statement.Id(functionName).Params(Id("id").String()).Parens(List(
		returns...,
	))
}

func generateGet(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderGetFunctionSignature(&Statement{}, props)

	implementation := renderGetFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("Get", props.Singular),
		List(Id("msg"), Err()).Op(":=").Id("s").Dot("crud").Dot("Read").Call(Id("id")),
		If(Err().Op("!=").Nil()).Block(
			Return(CBlock(CCode(true, Nil()), CCode(props.GetExists, Id("msg").Op("==").Nil()), CCode(true, Err()))...),
		),
		If(Id("msg").Op("==").Nil()).Block(
			Return(CBlock(CCode(true, Nil()), CCode(props.GetExists, False()), CCode(true, Nil()))...),
		),
		cast(props, Id(strings.ToLower(props.Singular)).Op(":=").Id("msg")),
		Return(CBlock(CCode(true, Id(strings.ToLower(props.Singular))), CCode(props.GetExists, True()), CCode(true, Nil()))...),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["get"] = generateGet
}
