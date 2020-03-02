package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
)

func renderAddFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Add%s", props.HumanName)
	return statement.Id(functionName).Params(Id(strings.ToLower(props.HumanName)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

// GenerateAdd generates the add method.
func GenerateAdd(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderAddFunctionSignature(&Statement{}, props)

	implementation := renderAddFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("Add", props.HumanName),
		Return(Id("s").Dot("underlying").Dot("Create").Call(Id(strings.ToLower(props.HumanName)))),
	)
	return interfaceMethod, implementation
}
