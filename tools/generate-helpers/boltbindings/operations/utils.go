package operations

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
)

// ConditionalCode represents a list of Jen Codes and the condition under which they will be included in a code block.
type ConditionalCode struct {
	codes     []Code
	condition bool
}

// CCode is a convenient way to create a ConditionalCode
func CCode(condition bool, codes ...Code) *ConditionalCode {
	return &ConditionalCode{codes: codes, condition: condition}
}

// CBlock (ConditionalBlock) takes a list of ConditionalCodes and returns the concatenated list of Codes for which the
// condition is true, in order
func CBlock(codes ...*ConditionalCode) []Code {
	var realized []Code
	for _, code := range codes {
		if code.condition {
			realized = append(realized, code.codes...)
		}
	}
	return realized
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

	implementation := sigFunc(common.RenderFuncSStarStore(), props).Block(
		common.RenderBoltMetricLine(crudCall, props.Singular),
		List(Id("_"), Id("_"), Err()).Op(":=").Id("s").Dot("crud").Dot(crudCall).Call(Id(argName)),
		Return(Err()),
	)

	return interfaceMethod, implementation
}

func renderUpdateUpsertMany(sigFunc func(*Statement, *GeneratorProperties) *Statement, props *GeneratorProperties, argName, crudCall string) (Code, Code) {
	interfaceMethod := sigFunc(&Statement{}, props)

	implementation := sigFunc(common.RenderFuncSStarStore(), props).Block(
		Id("msgs").Op(":=").Make(Index().Qual(packagenames.GogoProto, "Message"), Len(Id(argName))),
		For(
			List(Id("i"), Id("key")).Op(":=").Range().Id(argName).Block(
				Id("msgs").Index(Id("i")).Op("=").Id("key"),
			),
		),
		List(Id("_"), Id("_"), Err()).Op(":=").Id("s").Dot("crud").Dot(crudCall).Call(Id("msgs")),
		Return(Err()),
	)
	return interfaceMethod, implementation
}

// Use the props to cast the given statement to the type of the object we are trying to return
func cast(props *GeneratorProperties, statement *Statement) *Statement {
	return statement.Assert(Op("*").Qual(props.Pkg, props.Object))
}
