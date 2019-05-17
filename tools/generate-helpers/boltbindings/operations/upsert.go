package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

func renderUpsertFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Upsert%s", props.Singular)
	return statement.Id(functionName).Params(jen.Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateUpsert(props *GeneratorProperties) (jen.Code, jen.Code) {
	return renderAddUpdateUpsert(renderUpsertFunctionSignature, props, strings.ToLower(props.Singular), "Upsert")
}

func init() {
	supportedMethods["upsert"] = generateUpsert
}
