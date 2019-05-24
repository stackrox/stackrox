package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
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

	existsReturns := []Code{Id("storedKey")}
	nilReturns := []Code{Nil()}
	errReturns := []Code{Nil()}
	if props.GetExists {
		existsReturns = append(existsReturns, True())
		nilReturns = append(nilReturns, False())
		errReturns = append(errReturns, Id("msg").Op("==").Nil())
	}
	existsReturns = append(existsReturns, Nil())
	nilReturns = append(nilReturns, Nil())
	errReturns = append(errReturns, Err())

	implementation := renderGetFunctionSignature(renderFuncSStarStore(), props).Block(
		metricLine("Get", props.Singular),
		List(Id("msg"), Err()).Op(":=").Id("s").Dot("crud").Dot("Read").Call(Id("id")),
		If(Err().Op("!=").Nil()).Block(
			Return(errReturns...),
		),
		If(Id("msg").Op("==").Nil()).Block(
			Return(nilReturns...),
		),
		Id("storedKey").Op(":=").Id("msg").Assert(Op("*").Qual(props.Pkg, props.Object)),
		Return(existsReturns...),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["get"] = generateGet
}
