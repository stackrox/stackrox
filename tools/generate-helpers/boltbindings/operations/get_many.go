package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderGetManyFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Get%s", props.Plural)
	return statement.Id(functionName).Params(jen.Id("ids").Index().String()).Parens(jen.List(
		jen.Index().Op("*").Qual(props.Pkg, props.Object),
		jen.Error(),
	))
}

func generateGetMany(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderGetManyFunctionSignature(&jen.Statement{}, props)

	implementation := renderGetManyFunctionSignature(renderFuncSStarStore(), props).Block(
		jen.List(jen.Id("msgs"), jen.Id("err")).Op(":=").Id("s").Dot("crud").Dot("ReadBatch").Call(jen.Id("ids")),
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
	supportedMethods["get_many"] = generateGetMany
}
