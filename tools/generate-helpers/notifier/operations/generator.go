package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

type addAndNotifyGenerator struct {
	First string
	Third string
	Past  string
}

func (g *addAndNotifyGenerator) generate(props *GeneratorProperties) ([]Code, []Code, []Code) {
	var signatures []Code
	signatures = append(signatures, g.renderRegisterFunctionSignature(&Statement{}, props))
	signatures = append(signatures, g.renderCallFunctionSignature(&Statement{}, props))

	var implementations []Code
	implementations = append(implementations, g.renderRegisterImplementation(props))
	implementations = append(implementations, g.renderCallImplementation(props))

	variables := []Code{g.renderMemberVariable(props)}

	return signatures, variables, implementations
}

func (g *addAndNotifyGenerator) renderRegisterFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	funcName := fmt.Sprintf("On%s", g.First)
	valueToRegisterName := fmt.Sprintf("on%s", g.First)
	sig := statement.Id(funcName).Params(Id(valueToRegisterName).Func().Params(Id(strings.ToLower(props.Object)).Op("*").Qual(props.Pkg, props.Object)))
	return sig
}

func (g *addAndNotifyGenerator) renderCallFunctionSignature(statement *Statement, props *GeneratorProperties) *Statement {
	sig := statement.Id(g.Past).Params(Id(strings.ToLower(props.Object)).Op("*").Qual(props.Pkg, props.Object))
	return sig
}

func (g *addAndNotifyGenerator) renderRegisterImplementation(props *GeneratorProperties) Code {
	currentValuesFieldName := fmt.Sprintf("on%s", g.Third)
	addedValueFieldName := fmt.Sprintf("on%s", g.First)

	var blockContents []Code
	blockContents = append(blockContents, Id("n").Dot("lock").Dot("Lock").Call())
	blockContents = append(blockContents, Defer().Id("n").Dot("lock").Dot("Unlock").Call())

	blockContents = append(blockContents,
		Id("n").Dot(currentValuesFieldName).Op("=").Append(Id("n").Dot(currentValuesFieldName), Id(addedValueFieldName)),
	)
	return g.renderRegisterFunctionSignature(receiver(), props).Block(
		blockContents...,
	)
}

func (g *addAndNotifyGenerator) renderCallImplementation(props *GeneratorProperties) Code {
	fieldName := fmt.Sprintf("on%s", g.Third)

	var blockContents []Code
	blockContents = append(blockContents, Id("n").Dot("lock").Dot("RLock").Call())
	blockContents = append(blockContents, Defer().Id("n").Dot("lock").Dot("RUnlock").Call())

	blockContents = append(blockContents,
		For(List(Id("_"), Id("f")).Op(":=").Range().Id("n").Dot(fieldName)).Block(
			Id("f").Call(Id(strings.ToLower(props.Object))),
		),
	)
	return g.renderCallFunctionSignature(receiver(), props).Block(
		blockContents...,
	)
}

func (g *addAndNotifyGenerator) renderMemberVariable(props *GeneratorProperties) Code {
	fieldName := fmt.Sprintf("on%s", g.Third)
	return Id(fieldName).Index().Func().Params(Id(strings.ToLower(props.Object)).Op("*").Qual(props.Pkg, props.Object))
}

func receiver() *Statement {
	return Func().Params(Id("n").Op("*").Id("notifier"))
}
