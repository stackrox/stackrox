package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

func renderUpsertFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Upsert%s", props.Singular)
	return statement.Id(functionName).Params(Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateUpsert(props *GeneratorProperties) (Code, Code) {
	return renderUpdateUpsert(renderUpsertFunctionSignature, props, strings.ToLower(props.Singular), "Upsert")
}

func init() {
	supportedMethods["upsert"] = generateUpsert
}
