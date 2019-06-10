package operations

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

func renderDeleteFunctionSignature(statement *jen.Statement, props GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Delete%s", props.Singular)
	return statement.Id(functionName).Params(jen.Id("id").String()).Error()
}

func generateDelete(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderDeleteFunctionSignature(&jen.Statement{}, props)

	implementation := renderDeleteFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("Remove", props.Object),
		ifErrReturnError(jen.Id("b").Dot("index").Dot("Delete").Call(jen.Id("id"))),
		jen.Return(incrementTxnCount()),
	)

	return interfaceMethod, implementation
}

func renderDeleteManyFunctionSignature(statement *jen.Statement, props GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Delete%s", props.Plural)
	return statement.Id(functionName).Params(jen.Id("ids").Index().String()).Error()
}

func generateDeleteMany(props GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderDeleteManyFunctionSignature(&jen.Statement{}, props)

	implementation := renderDeleteManyFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("RemoveMany", props.Object),
		jen.Id("batch").Op(":=").Id("b").Dot("index").Dot("NewBatch").Params(),
		jen.For(jen.List(jen.Op("_"), jen.Id("id")).Op(":=").Range().Id("ids")).Block(
			jen.Id("batch").Dot("Delete").Call(jen.Id("id")),
		),
		ifErrReturnError(bIndex().Dot("Batch").Call(jen.Id("batch"))),
		jen.Return(incrementTxnCount()),
	)

	return interfaceMethod, implementation

}

func init() {
	supportedMethods["delete"] = generateDelete
	supportedMethods["deleteMany"] = generateDeleteMany
}
