package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/boltbindings/packagenames"
)

func renderAddFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Add%s", props.Singular)
	sig := statement.Id(functionName).Params(jen.Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object))
	if props.IDField != "" {
		sig.Parens(jen.List(jen.String(), jen.Error()))
	} else {
		sig.Error()
	}
	return sig
}

func generateAdd(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderAddFunctionSignature(&jen.Statement{}, props)

	var blockContents []jen.Code
	var returnContents []jen.Code
	blockContents = append(blockContents, metricLine("Add", props.Singular))
	if props.IDField != "" {
		blockContents = append(blockContents, jen.Id("newId").Op(":=").Qual(packagenames.UUID, "NewV4").Call().Dot("String").Call())
		blockContents = append(blockContents, jen.Id(strings.ToLower(props.Singular)).Dot(props.IDField).Op("=").Id("newId"))
		returnContents = append(returnContents, jen.Id("newId"))
	}
	returnContents = append(returnContents, jen.Id("s").Dot("crud").Dot("Create").Call(jen.Id(strings.ToLower(props.Singular))))
	blockContents = append(blockContents, jen.Return(returnContents...))
	implementation := renderAddFunctionSignature(renderFuncSStarStore(), props).Block(
		blockContents...,
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add"] = generateAdd
}
