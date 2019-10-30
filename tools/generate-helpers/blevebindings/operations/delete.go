package operations

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
)

func renderDeleteFunctionSignature(statement *Statement, props GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Delete%s", props.Singular)
	return statement.Id(functionName).Params(Id("id").String()).Error()
}

func generateDelete(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderDeleteFunctionSignature(&Statement{}, props)

	implementation := renderDeleteFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("Remove", props.Object),
		ifErrReturnError(Id("b").Dot("index").Dot("Delete").Call(Id("id"))),
		Return(incrementTxnCount(props.NeedsTxManager)),
	)

	return interfaceMethod, implementation
}

func renderDeleteManyFunctionSignature(statement *Statement, props GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Delete%s", props.Plural)
	return statement.Id(functionName).Params(Id("ids").Index().String()).Error()
}

func generateDeleteMany(props GeneratorProperties) (Code, Code) {
	interfaceMethod := renderDeleteManyFunctionSignature(&Statement{}, props)

	implementation := renderDeleteManyFunctionSignature(renderFuncBStarIndexer(), props).Block(
		metricLine("RemoveMany", props.Object),
		Id("batch").Op(":=").Id("b").Dot("index").Dot("NewBatch").Params(),
		For(List(Op("_"), Id("id")).Op(":=").Range().Id("ids")).Block(
			Id("batch").Dot("Delete").Call(Id("id")),
		),
		ifErrReturnError(bIndex().Dot("Batch").Call(Id("batch"))),
		Return(incrementTxnCount(props.NeedsTxManager)),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["delete"] = generateDelete
	supportedMethods["deleteMany"] = generateDeleteMany
}
