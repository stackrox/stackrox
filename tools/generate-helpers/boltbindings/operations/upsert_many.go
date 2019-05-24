package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

func renderUpsertManyFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Upsert%s", props.Plural)
	return statement.Id(functionName).
		Params(Id(strings.ToLower(props.Plural)).
			Index().Op("*").Qual(props.Pkg, props.Object),
		).
		Error()
}

func generateUpsertMany(props *GeneratorProperties) (Code, Code) {
	return renderUpdateUpsertMany(renderUpsertManyFunctionSignature, props, strings.ToLower(props.Plural), "UpsertBatch")
}

func init() {
	supportedMethods["upsert_many"] = generateUpsertMany
}
