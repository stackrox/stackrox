package operations

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/boltbindings/packagenames"
)

func renderFuncSStarStore() *Statement {
	return Func().Params(Id("s").Op("*").Id("store"))
}

func renderIfErrReturnNilErr(extraResults ...Code) Code {
	allResults := make([]Code, 0, len(extraResults)+2)
	allResults = append(allResults, Nil())
	allResults = append(allResults, extraResults...)
	allResults = append(allResults, Err())
	return If(Err().Op("!=").Nil()).Block(
		Return(allResults...),
	)
}

func renderUpdateUpsert(sigFunc func(*Statement, *GeneratorProperties) *Statement, props *GeneratorProperties, argName, crudCall string) (Code, Code) {
	interfaceMethod := sigFunc(&Statement{}, props)

	implementation := sigFunc(renderFuncSStarStore(), props).Block(
		metricLine(crudCall, props.Singular),
		Return(Id("s").Dot("crud").Dot(crudCall).Call(Id(argName))),
	)

	return interfaceMethod, implementation
}

func renderUpdateUpsertMany(sigFunc func(*Statement, *GeneratorProperties) *Statement, props *GeneratorProperties, argName, crudCall string) (Code, Code) {
	interfaceMethod := sigFunc(&Statement{}, props)

	implementation := sigFunc(renderFuncSStarStore(), props).Block(
		Id("msgs").Op(":=").Make(Index().Qual(packagenames.GogoProto, "Message"), Len(Id(argName))),
		For(
			List(Id("i"), Id("key")).Op(":=").Range().Id(argName).Block(
				Id("msgs").Index(Id("i")).Op("=").Id("key"),
			),
		),
		Return(Id("s").Dot("crud").Dot(crudCall).Call(Id("msgs"))),
	)
	return interfaceMethod, implementation
}

func metricLine(op, name string) *Statement {
	return Defer().Qual(packagenames.Metrics, "SetBoltOperationDurationTime").Call(Qual("time", "Now").Call(), Qual(packagenames.Ops, op), Lit(name))
}
