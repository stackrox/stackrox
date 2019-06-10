package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/blevebindings/packagenames"
)

func renderAddManyFunctionSignature(statement *jen.Statement, props GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Add%s", props.Plural)
	return statement.Id(functionName).Params(jen.Id(strings.ToLower(props.Plural)).Index().Op("*").Qual(props.Pkg, props.Object)).Error()
}

func generateAddMany(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderAddManyFunctionSignature(&jen.Statement{}, props)
	wrapperType := MakeWrapperType(props.Object)

	implementation := renderAddManyFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("AddMany", props.Object),
		jen.Id("batchManager").Op(":=").Qual(packagenames.RoxBatcher, "New").Call(jen.Id("len").Call(jen.Id(strings.ToLower(props.Plural))), jen.Id("batchSize")),
		jen.For().Block(
			jen.List(jen.Id("start"), jen.Id("end"), jen.Id("ok")).Op(":=").Id("batchManager").Dot("Next").Call(),
			jen.If(jen.Op("!").Id("ok")).Block(
				jen.Op("break"),
			),
			jen.If(jen.Err().Op(":=").Id("b").Dot("processBatch").Call(jen.Id(strings.ToLower(props.Plural)).Index(jen.Id("start").Op(":").Id("end"))), jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Err()),
			),
		),
		jen.Return(incrementTxnCount()),
	).Line().Line().Func().Params(jen.Id("b").Op("*").Id("indexerImpl")).Id("processBatch").Params(jen.Id(strings.ToLower(props.Plural)).Index().Op("*").Qual(props.Pkg, props.Object)).Error().Block(
		jen.Id("batch").Op(":=").Id("b").Dot("index").Dot("NewBatch").Params(),
		jen.For(jen.List(jen.Id("_"), jen.Id(strings.ToLower(props.Singular))).Op(":=").Range().Id(strings.ToLower(props.Plural))).Block(
			jen.If(jen.Err().Op(":=").Id("batch").Dot("Index").Call(
				jen.Id(strings.ToLower(props.Singular)).Dot("GetId").Call(),
				jen.Op("&").Id(wrapperType).Values(jen.Dict{
					jen.Id("Type"):       jen.Qual(packagenames.V1, props.SearchCategory).Dot("String").Call(),
					jen.Id(props.Object): jen.Id(strings.ToLower(props.Singular)),
				}),
			), jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Err()),
			),
		),
		jen.Return(jen.Id("b").Dot("index").Dot("Batch").Call(jen.Id("batch"))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add_multiple"] = generateAddMany
}
