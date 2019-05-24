package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

func renderUpdateFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Update%s", props.Singular)
	return statement.Id(functionName).Params(jen.Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateUpdate(props *GeneratorProperties) (jen.Code, jen.Code) {
	return renderUpdateUpsert(renderUpdateFunctionSignature, props, strings.ToLower(props.Singular), "Update")
}

func init() {
	supportedMethods["update"] = generateUpdate
}
