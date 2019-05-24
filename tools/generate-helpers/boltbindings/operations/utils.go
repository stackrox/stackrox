package operations

import (
	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/boltbindings/packagenames"
)

func renderFuncSStarStore() *jen.Statement {
	return jen.Func().Params(jen.Id("s").Op("*").Id("store"))
}

func renderIfErrReturnNilErr() jen.Code {
	return jen.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
}

func renderUpdateUpsert(sigFunc func(*jen.Statement, *GeneratorProperties) *jen.Statement, props *GeneratorProperties, argName, crudCall string) (jen.Code, jen.Code) {
	interfaceMethod := sigFunc(&jen.Statement{}, props)

	implementation := sigFunc(renderFuncSStarStore(), props).Block(
		metricLine(crudCall, props.Singular),
		jen.Return(jen.Id("s").Dot("crud").Dot(crudCall).Call(jen.Id(argName))),
	)

	return interfaceMethod, implementation
}

func renderUpdateUpsertMany(sigFunc func(*jen.Statement, *GeneratorProperties) *jen.Statement, props *GeneratorProperties, argName, crudCall string) (jen.Code, jen.Code) {
	interfaceMethod := sigFunc(&jen.Statement{}, props)

	implementation := sigFunc(renderFuncSStarStore(), props).Block(
		jen.Id("msgs").Op(":=").Make(jen.Index().Qual(packagenames.GogoProto, "Message"), jen.Len(jen.Id(argName))),
		jen.For(
			jen.List(jen.Id("i"), jen.Id("key")).Op(":=").Range().Id(argName).Block(
				jen.Id("msgs").Index(jen.Id("i")).Op("=").Id("key"),
			),
		),
		jen.Return(jen.Id("s").Dot("crud").Dot(crudCall).Call(jen.Id("msgs"))),
	)
	return interfaceMethod, implementation
}

func metricLine(op, name string) *jen.Statement {
	return jen.Defer().Qual(packagenames.Metrics, "SetBoltOperationDurationTime").Call(jen.Qual("time", "Now").Call(), jen.Qual(packagenames.Ops, op), jen.Lit(name))
}
