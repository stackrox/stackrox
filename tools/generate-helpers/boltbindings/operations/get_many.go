package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
)

func renderGetManyFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Get%s", props.Plural)
	return statement.Id(functionName).Params(Id("ids").Index().String()).Parens(List(
		Index().Op("*").Qual(props.Pkg, props.Object),
		Index().Int(),
		Error(),
	))
}

func generateGetMany(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderGetManyFunctionSignature(&Statement{}, props)

	implementation := renderGetManyFunctionSignature(renderFuncSStarStore(), props).Block(
		If(Len(Id("ids")).Op("==").Lit(0)).Block(
			Return(Nil(), Nil(), Nil()),
		),
		metricLine("GetMany", props.Singular),
		List(Id("msgs"), Id("missingIndices"), Id("err")).Op(":=").Id("s").Dot("crud").Dot("ReadBatch").Call(Id("ids")),
		renderIfErrReturnNilErr(Nil()),
		Id("storedKeys").Op(":=").Make(Index().Op("*").Qual(props.Pkg, props.Object), Len(Id("msgs"))),
		For(List(Id("i"), Id("msg")).Op(":=").Range().Id("msgs")).Block(
			Id("storedKeys").Index(Id("i")).Op("=").Id("msg").Assert(Op("*").Qual(props.Pkg, props.Object)),
		),
		Return(Id("storedKeys"), Id("missingIndices"), Nil()),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["get_many"] = generateGetMany
}
