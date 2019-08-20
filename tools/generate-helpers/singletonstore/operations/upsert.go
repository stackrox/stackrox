package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
)

func renderUpsertFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Upsert%s", props.HumanName)
	return statement.Id(functionName).Params(Id(strings.ToLower(props.HumanName)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

// GenerateUpsert generates the upsert method.
func GenerateUpsert(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderUpsertFunctionSignature(&Statement{}, props)

	implementation := renderUpsertFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine("Upsert", props.HumanName),
		Return(Id("s").Dot("underlying").Dot("Upsert").Call(Id(strings.ToLower(props.HumanName)))),
	)
	return interfaceMethod, implementation
}
