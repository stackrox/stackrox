package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/stackrox/tools/generate-helpers/common"
)

func renderListFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("List%s", props.Plural)
	return statement.Id(functionName).Params().Parens(List(
		Index().Op("*").Qual(props.Pkg, props.Object),
		Error(),
	))
}

func generateList(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderListFunctionSignature(&Statement{}, props)

	implementation := renderListFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("GetAll", props.Singular),
		List(Id("msgs"), Err()).Op(":=").Id("s").Dot("crud").Dot("ReadAll").Call(),
		renderIfErrReturnNilErr(),
		Id("storedKeys").Op(":=").Make(Index().Op("*").Qual(props.Pkg, props.Object), Len(Id("msgs"))),
		For(List(Id("i"), Id("msg")).Op(":=").Range().Id("msgs")).Block(
			cast(props, Id("storedKeys").Index(Id("i")).Op("=").Id("msg")),
		),
		Return(Id("storedKeys"), Nil()),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["list"] = generateList
}
