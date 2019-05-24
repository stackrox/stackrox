package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderListFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("List%s", props.Plural)
	return statement.Id(functionName).Params().Parens(jen.List(
		jen.Index().Op("*").Qual(props.Pkg, props.Object),
		jen.Error(),
	))
}

func generateList(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderListFunctionSignature(&jen.Statement{}, props)

	implementation := renderListFunctionSignature(renderFuncSStarStore(), props).Block(
		metricLine("GetAll", props.Singular),
		jen.List(jen.Id("msgs"), jen.Err()).Op(":=").Id("s").Dot("crud").Dot("ReadAll").Call(),
		renderIfErrReturnNilErr(),
		jen.Id("storedKeys").Op(":=").Make(jen.Index().Op("*").Qual(props.Pkg, props.Object), jen.Len(jen.Id("msgs"))),
		jen.For(jen.List(jen.Id("i"), jen.Id("msg")).Op(":=").Range().Id("msgs")).Block(
			jen.Id("storedKeys").Index(jen.Id("i")).Op("=").Id("msg").Assert(jen.Op("*").Qual(props.Pkg, props.Object)),
		),
		jen.Return(jen.Id("storedKeys"), jen.Nil()),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["list"] = generateList
}
