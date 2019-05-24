package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

func renderUpdateManyFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Update%s", props.Plural)
	return statement.Id(functionName).
		Params(Id(strings.ToLower(props.Plural)).
			Index().Op("*").Qual(props.Pkg, props.Object),
		).
		Error()
}

func generateUpdateMany(props *GeneratorProperties) (Code, Code) {
	return renderUpdateUpsertMany(renderUpdateManyFunctionSignature, props, strings.ToLower(props.Plural), "UpdateBatch")
}

func init() {
	supportedMethods["update_many"] = generateUpdateMany
}
