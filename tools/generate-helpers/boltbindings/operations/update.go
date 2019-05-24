package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

func renderUpdateFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Update%s", props.Singular)
	return statement.Id(functionName).Params(Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateUpdate(props *GeneratorProperties) (Code, Code) {
	return renderUpdateUpsert(renderUpdateFunctionSignature, props, strings.ToLower(props.Singular), "Update")
}

func init() {
	supportedMethods["update"] = generateUpdate
}
