package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
)

func renderCountFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Count%s", props.Plural)
	return statement.Id(functionName).Params().Parens(List(Id("count").Int(), Err().Error()))
}

func generateCount(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderCountFunctionSignature(&Statement{}, props)

	implementation := renderCountFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("Count", props.Singular),
		Return(Id("s").Dot("crud").Dot("Count").Call()),
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["count"] = generateCount
}
