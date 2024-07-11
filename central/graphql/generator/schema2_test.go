package generator

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/assert"
)

func assertSchemaSame(t *testing.T, expected, actual string) {
	exp := scanner.Scanner{}
	act := scanner.Scanner{}
	exp.Filename = "<expected>"
	act.Filename = "<actual>"
	exp.Init(strings.NewReader(expected))
	act.Init(strings.NewReader(actual))
	for r := exp.Scan(); r != scanner.EOF; r = exp.Scan() {
		if act.Scan() == scanner.EOF {
			t.Errorf("%q at %s but EOF at %s", exp.TokenText(), exp.Pos(), act.Pos())
			return
		}
		if exp.TokenText() != act.TokenText() {
			t.Errorf("%q at %s but %q at %s", exp.TokenText(), exp.Pos(), act.TokenText(), act.Pos())
			return
		}
	}
	if act.Scan() != scanner.EOF {
		t.Errorf("EOF at %s but %q at %s", exp.Pos(), act.TokenText(), act.Pos())
	}
}

func TestSchemaBuilderImpl_AddQuery(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddQuery("hello: String"))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema { query: Query }
type Query { hello: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddEnumType(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddEnumType("Enum", []string{"A", "B", "C"}))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
enum Enum {A B C}
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddInterfaceType(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddInterfaceType("Interface", []string{"a: String"}))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
interface Interface { a: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddUnionType(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddType("A", []string{"a: String"}))
	assert.NoError(t, b.AddType("B", []string{"b: String"}))
	assert.NoError(t, b.AddUnionType("C", []string{"A", "B"}))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
type A { a: String }
type B { b: String }
union C = A | B
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddType(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddType("T", []string{"t: String"}))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
type T { t: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddTypeWithInterface(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddType("T", []string{"t: String"}, "I"))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
type T implements I { t: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddTypeWithMultipleInterfaces(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddType("T", []string{"t: String"}, "I1", "I2"))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
type T implements I1 & I2 { t: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddInput(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddInput("I", []string{"i: String"}))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
input I { i: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddExtraResolver(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddType("T", []string{"t: String"}))
	assert.NoError(t, b.AddExtraResolver("T", "extra: String"))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema {}
type T {
	t: String
	extra: String
}
scalar Time
`
	assertSchemaSame(t, expected, actual)
}

func TestSchemaBuilderImpl_AddMutation(t *testing.T) {
	b := NewSchemaBuilder()
	assert.NoError(t, b.AddInput("I", []string{"i: String"}))
	assert.NoError(t, b.AddMutation("mut(i: I): String"))
	actual, err := b.Render()
	assert.NoError(t, err)
	expected := `
schema { mutation: Mutation }
type Mutation { mut(i: I): String }
input I { i: String }
scalar Time
`
	assertSchemaSame(t, expected, actual)
}
