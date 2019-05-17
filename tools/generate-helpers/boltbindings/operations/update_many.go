package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

func renderUpdateManyFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Update%s", props.Plural)
	return statement.Id(functionName).
		Params(jen.Id(strings.ToLower(props.Plural)).
			Index().Op("*").Qual(props.Pkg, props.Object),
		).
		Error()
}

func generateUpdateMany(props *GeneratorProperties) (jen.Code, jen.Code) {
	return renderAddUpdateUpsertMany(renderUpdateManyFunctionSignature, props, strings.ToLower(props.Plural), "UpdateBatch")
}

func init() {
	supportedMethods["update_many"] = generateUpdateMany
}
