package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/stackrox/tools/generate-helpers/common"
	"github.com/stackrox/stackrox/tools/generate-helpers/common/packagenames"
)

func renderAddManyFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	functionName := fmt.Sprintf("Add%s", props.Plural)
	sig := statement.Id(functionName).
		Params(Id(strings.ToLower(props.Plural)).
			Index().Op("*").Qual(props.Pkg, props.Object),
		)
	if props.IDField != "" {
		sig.Parens(List(Index().String(), Error()))
	} else {
		sig.Error()
	}
	return sig
}

func generateAddMany(props *GeneratorProperties) (Code, Code) {
	interfaceMethod := renderAddManyFunctionSignature(&Statement{}, props)
	var outerBlock []Code
	var innerBlock []Code
	var returnBlock []Code
	outerBlock = append(outerBlock, common.RenderBoltMetricLine("AddMany", props.Singular))
	if props.IDField != "" {
		outerBlock = append(outerBlock, Id("newIds").Op(":=").Make(Index().String(), Len(Id(strings.ToLower(props.Plural)))))
		innerBlock = append(innerBlock, Id("newId").Op(":=").Qual(packagenames.UUID, "NewV4").Call().Dot("String").Call())
		innerBlock = append(innerBlock, Id("key").Dot(props.IDField).Op("=").Id("newId"))
		innerBlock = append(innerBlock, Id("newIds").Index(Id("i")).Op("=").Id("newId"))
		returnBlock = append(returnBlock, Id("newIds"))
	}
	returnBlock = append(returnBlock, Id("s").Dot("crud").Dot("CreateBatch").Call(Id("msgs")))
	innerBlock = append(innerBlock, Id("msgs").Index(Id("i")).Op("=").Id("key"))
	outerBlock = append(outerBlock, Id("msgs").Op(":=").Make(Index().Qual(packagenames.GogoProto, "Message"), Len(Id(strings.ToLower(props.Plural)))))
	outerBlock = append(outerBlock, For(
		List(Id("i"), Id("key")).Op(":=").Range().Id(strings.ToLower(props.Plural)).Block(
			innerBlock...,
		),
	))
	outerBlock = append(outerBlock, Return(returnBlock...))

	implementation := renderAddManyFunctionSignature(common.RenderFuncSStarStore(), props).Block(
		outerBlock...,
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add_many"] = generateAddMany
}
