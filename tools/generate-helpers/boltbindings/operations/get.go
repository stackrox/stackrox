package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderGetFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Get%s", props.Singular)
	returns := []jen.Code{jen.Op("*").Qual(props.Pkg, props.Object)}
	if props.GetExists {
		returns = append(returns, jen.Bool())
	}
	returns = append(returns, jen.Error())
	return statement.Id(functionName).Params(jen.Id("id").String()).Parens(jen.List(
		returns...,
	))
}

func generateGet(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderGetFunctionSignature(&jen.Statement{}, props)

	existsReturns := []jen.Code{jen.Id("storedKey")}
	nilReturns := []jen.Code{jen.Nil()}
	errReturns := []jen.Code{jen.Nil()}
	if props.GetExists {
		existsReturns = append(existsReturns, jen.True())
		nilReturns = append(nilReturns, jen.False())
		errReturns = append(errReturns, jen.Id("msg").Op("==").Nil())
	}
	existsReturns = append(existsReturns, jen.Nil())
	nilReturns = append(nilReturns, jen.Nil())
	errReturns = append(errReturns, jen.Err())

	implementation := renderGetFunctionSignature(renderFuncSStarStore(), props).Block(
		jen.List(jen.Id("msg"), jen.Err()).Op(":=").Id("s").Dot("crud").Dot("Read").Call(jen.Id("id")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(errReturns...),
		),
		jen.If(jen.Id("msg").Op("==").Nil()).Block(
			jen.Return(nilReturns...),
		),
		jen.Id("storedKey").Op(":=").Id("msg").Assert(jen.Op("*").Qual(props.Pkg, props.Object)),
		jen.Return(existsReturns...),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["get"] = generateGet
}
