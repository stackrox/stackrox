package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/stackrox/tools/generate-helpers/common/packagenames"
)

func renderAddManyFunctionSignature(statement *Statement, props GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Add%s", props.Plural)
	return statement.Id(functionName).Params(Id(strings.ToLower(props.Plural)).Index().Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateAddMany(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderAddManyFunctionSignature(&Statement{}, props)
	wrapperType := MakeWrapperType(props.Object)

	implementation := renderAddManyFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("AddMany", props.Object),
		Id("batchManager").Op(":=").Qual(packagenames.RoxBatcher, "New").Call(Id("len").Call(Id(strings.ToLower(props.Plural))), Id("batchSize")),
		For().Block(
			List(Id("start"), Id("end"), Id("ok")).Op(":=").Id("batchManager").Dot("Next").Call(),
			If(Op("!").Id("ok")).Block(
				Op("break"),
			),
			If(Err().Op(":=").Id("b").Dot("processBatch").Call(Id(strings.ToLower(props.Plural)).Index(Id("start").Op(":").Id("end"))), Err().Op("!=").Nil()).Block(
				Return(Err()),
			),
		),
		Return(Nil()),
	).Line().Line().Func().Params(Id("b").Op("*").Id("indexerImpl")).Id("processBatch").Params(Id(strings.ToLower(props.Plural)).Index().Op("*").Qual(props.Pkg, props.Object)).Error().Block(
		Id("batch").Op(":=").Id("b").Dot("index").Dot("NewBatch").Params(),
		For(List(Id("_"), Id(strings.ToLower(props.Singular))).Op(":=").Range().Id(strings.ToLower(props.Plural))).Block(
			If(Err().Op(":=").Id("batch").Dot("Index").Call(
				Id(strings.ToLower(props.Singular)).Dot(props.IDFunc).Call(),
				Op("&").Id(wrapperType).Values(Dict{
					Id("Type"):       Qual(packagenames.V1, props.SearchCategory).Dot("String").Call(),
					Id(props.Object): Id(strings.ToLower(props.Singular)),
				}),
			), Err().Op("!=").Nil()).Block(
				Return(Err()),
			),
		),
		Return(Id("b").Dot("index").Dot("Batch").Call(Id("batch"))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add_multiple"] = generateAddMany
}
