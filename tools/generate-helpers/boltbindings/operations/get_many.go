package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
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
	storedKeys := "storedKeys"
	interfaceMethod := renderGetManyFunctionSignature(&Statement{}, props)

	implementation := renderGetManyFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		If(Len(Id("ids")).Op("==").Lit(0)).Block(
			Return(Nil(), Nil(), Nil()),
		),
		common.RenderBoltMetricLine("GetMany", props.Singular),
		List(Id("msgs"), Id("missingIndices"), Err()).Op(":=").Id("s").Dot("crud").Dot("ReadBatch").Call(Id("ids")),
		renderIfErrReturnNilErr(Nil()),
		Id(storedKeys).Op(":=").Make(Index().Op("*").Qual(props.Pkg, props.Object), Lit(0), Len(Id("msgs"))),
		For(List(Id("_"), Id("msg")).Op(":=").Range().Id("msgs")).Block(
			Id(storedKeys).Op("=").Append(Id(storedKeys), cast(props, Id("msg"))),
		),
		Return(Id(storedKeys), Id("missingIndices"), Nil()),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["get_many"] = generateGetMany
}
