package operations

import (
	. "github.com/dave/jennifer/jen"
)

func renderNeedsInitialIndexingFunctionSignature(statement *Statement) *Statement {
	functionName := "NeedsInitialIndexing"
	return statement.Id(functionName).Params().Parens(List(Bool(), Error()))
}

func generateNeedsInitialIndexing(_ GeneratorProperties) (Code, Code) {
	interfaceMethod := renderNeedsInitialIndexingFunctionSignature(&Statement{})

	implementation := renderNeedsInitialIndexingFunctionSignature(renderFuncBStarIndexer()).Block(
		List(Id("data"), Err()).Op(":=").Id("b").Dot("index").Dot("GetInternal").Call(Id("[]byte(resourceName)")),
		If(Err().Op("!=").Nil()).Block(
			Return(List(False(), Err())),
		),
		Return(List(Op("!").Qual("bytes", "Equal").Call(List(Id(`[]byte("old")`), Id("data"))), Nil())),
	)

	return interfaceMethod, implementation
}

func renderMarkInitialIndexingFunctionSignature(statement *Statement) *Statement {
	functionName := "MarkInitialIndexingComplete"
	return statement.Id(functionName).Params().Error()
}

func generateMarkInitialIndexing(_ GeneratorProperties) (Code, Code) {
	interfaceMethod := renderMarkInitialIndexingFunctionSignature(&Statement{})

	implementation := renderMarkInitialIndexingFunctionSignature(renderFuncBStarIndexer()).Block(
		Return(Id("b").Dot("index").Dot("SetInternal").Call(Id("[]byte(resourceName)"), Id(`[]byte("old")`))),
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["needsInitialIndexing"] = generateNeedsInitialIndexing
	supportedMethods["markInitialIndexing"] = generateMarkInitialIndexing
}
