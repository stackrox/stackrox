package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
)

func renderGetManyFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Get%s", props.Plural)
	return statement.Id(functionName).Params(Id("ids").Index().String()).Parens(List(
		Index().Op("*").Qual(props.Pkg, props.Object),
		Error(),
	))
}

func generateGetMany(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderGetManyFunctionSignature(&Statement{}, props)

	implementation := renderGetManyFunctionSignature(renderFuncSStarStore(), props).Block(
		metricLine("GetMany", props.Singular),
		List(Id("msgs"), Id("err")).Op(":=").Id("s").Dot("crud").Dot("ReadBatch").Call(Id("ids")),
		renderIfErrReturnNilErr(),
		Id("storedKeys").Op(":=").Make(Index().Op("*").Qual(props.Pkg, props.Object), Len(Id("msgs"))),
		For(List(Id("i"), Id("msg")).Op(":=").Range().Id("msgs")).Block(
			Id("storedKeys").Index(Id("i")).Op("=").Id("msg").Assert(Op("*").Qual(props.Pkg, props.Object)),
		),
		Return(Id("storedKeys"), Nil()),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["get_many"] = generateGetMany
}
