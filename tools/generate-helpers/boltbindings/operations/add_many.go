package operations

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/boltbindings/packagenames"
)

func renderAddManyFunctionSignature(statement *jen.Statement, props *GeneratorProperties) *jen.Statement {
	functionName := fmt.Sprintf("Add%s", props.Plural)
	sig := statement.Id(functionName).
		Params(jen.Id(strings.ToLower(props.Plural)).
			Index().Op("*").Qual(props.Pkg, props.Object),
		)
	if props.IDField != "" {
		sig.Parens(jen.List(jen.Index().String(), jen.Error()))
	} else {
		sig.Error()
	}
	return sig
}

func generateAddMany(props *GeneratorProperties) (jen.Code, jen.Code) {
	interfaceMethod := renderAddManyFunctionSignature(&jen.Statement{}, props)
	var outerBlock []jen.Code
	var innerBlock []jen.Code
	var returnBlock []jen.Code
	if props.IDField != "" {
		outerBlock = append(outerBlock, jen.Id("newIds").Op(":=").Make(jen.Index().String(), jen.Len(jen.Id(strings.ToLower(props.Plural)))))
		innerBlock = append(innerBlock, jen.Id("newId").Op(":=").Qual(packagenames.UUID, "NewV4").Call().Dot("String").Call())
		innerBlock = append(innerBlock, jen.Id("key").Dot(props.IDField).Op("=").Id("newId"))
		innerBlock = append(innerBlock, jen.Id("newIds").Index(jen.Id("i")).Op("=").Id("newId"))
		returnBlock = append(returnBlock, jen.Id("newIds"))
	}
	returnBlock = append(returnBlock, jen.Id("s").Dot("crud").Dot("CreateBatch").Call(jen.Id("msgs")))
	innerBlock = append(innerBlock, jen.Id("msgs").Index(jen.Id("i")).Op("=").Id("key"))
	outerBlock = append(outerBlock, jen.Id("msgs").Op(":=").Make(jen.Index().Qual(packagenames.GogoProto, "Message"), jen.Len(jen.Id(strings.ToLower(props.Plural)))))
	outerBlock = append(outerBlock, jen.For(
		jen.List(jen.Id("i"), jen.Id("key")).Op(":=").Range().Id(strings.ToLower(props.Plural)).Block(
			innerBlock...,
		),
	))
	outerBlock = append(outerBlock, jen.Return(returnBlock...))

	implementation := renderAddManyFunctionSignature(renderFuncSStarStore(), props).Block(
		outerBlock...,
	)
	return interfaceMethod, implementation
}

func init() {
	supportedMethods["add_many"] = generateAddMany
}
