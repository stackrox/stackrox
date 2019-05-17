package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

func renderUpsertManyFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Upsert%s", props.Plural)
	return statement.Id(functionName).
		Params(jen.Id(strings.ToLower(props.Plural)).
			Index().Op("*").Qual(props.Pkg, props.Object),
		).
		Error()
}

func generateUpsertMany(props *GeneratorProperties) (jen.Code, jen.Code) {
	return renderAddUpdateUpsertMany(renderUpsertManyFunctionSignature, props, strings.ToLower(props.Plural), "UpsertBatch")
}

func init() {
	supportedMethods["upsert_many"] = generateUpsertMany
}
