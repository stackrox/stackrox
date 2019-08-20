package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
)

func renderAddFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Add%s", props.Singular)
	sig := statement.Id(functionName).Params(Id(strings.ToLower(props.Singular)).Op("*").Qual(props.Pkg, props.Object))
	if props.IDField != "" {
		sig.Parens(List(String(), Error()))
	} else {
		sig.Error()
	}
	return sig
}

func generateAdd(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderAddFunctionSignature(&Statement{}, props)

	var blockContents []Code
	var returnContents []Code
	blockContents = append(blockContents, common.RenderBoltMetricLine("Add", props.Singular))
	if props.IDField != "" {
		blockContents = append(blockContents, Id("newId").Op(":=").Qual(packagenames.UUID, "NewV4").Call().Dot("String").Call())
		blockContents = append(blockContents, Id(strings.ToLower(props.Singular)).Dot(props.IDField).Op("=").Id("newId"))
		returnContents = append(returnContents, Id("newId"))
	}
	returnContents = append(returnContents, Id("s").Dot("crud").Dot("Create").Call(Id(strings.ToLower(props.Singular))))
	blockContents = append(blockContents, Return(returnContents...))
	implementation := renderAddFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		blockContents...,
	)

	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add"] = generateAdd
}
