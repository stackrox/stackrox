package celcompile

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/interpreter/functions"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

//A very simple example of using user defined structures with instance functions with cel-go

import (
	"log"
	"testing"
)

// our custom type
type Test struct {
	a, b int
}

// operations supported by the custom type
func (t *Test) Add() int {
	return t.a + t.b
}
func (t *Test) Subtract() int {
	return t.a - t.b
}

// the CEL type to represent Test
var TestType = types.NewTypeValue("Test", traits.ReceiverType)

func (t Test) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	panic("not required")
}

func (t Test) ConvertToType(typeVal ref.Type) ref.Val {
	panic("not required")
}

func (t Test) Equal(other ref.Val) ref.Val {

	o, ok := other.Value().(Test)
	if ok {
		if o == t {
			return types.Bool(true)
		} else {
			return types.Bool(false)
		}
	} else {
		return types.ValOrErr(other, "%v is not of type Test", other)
	}
}

func (t Test) Type() ref.Type {
	return TestType
}

func (t Test) Value() interface{} {
	return t
}

func (t Test) Receive(function string, overload string, args []ref.Val) ref.Val {

	if function == "Add" {
		return types.Int(t.Add())
	} else if function == "Subtract" {
		return types.Int(t.Subtract())
	}
	return types.ValOrErr(TestType, "no such function - %s", function)
}

func (t *Test) HasTrait(trait int) bool {
	return trait == traits.ReceiverType
}

func (t *Test) TypeName() string {
	return TestType.TypeName()
}

type customTypeAdapter struct {
}

func (customTypeAdapter) NativeToValue(value interface{}) ref.Val {
	val, ok := value.(Test)
	if ok {
		return val
	} else {
		//let the default adapter handle other cases
		return types.DefaultTypeAdapter.NativeToValue(value)
	}
}

func TestExprEval_CelGo(t *testing.T) {
	env, err := cel.NewEnv(cel.CustomTypeAdapter(&customTypeAdapter{}),
		cel.Declarations(
			decls.NewIdent("test", decls.NewObjectType("Test"), nil),
			decls.NewFunction("MulBy3",
				decls.NewOverload("mulby3_int", []*expr.Type{decls.Int}, decls.Int)),
			decls.NewFunction("Add",
				decls.NewInstanceOverload("test_add", []*expr.Type{decls.NewObjectType("Test")}, decls.Int)),
			decls.NewFunction("Subtract",
				decls.NewInstanceOverload("test_subtract", []*expr.Type{decls.NewObjectType("Test")}, decls.Int))))
	if err != nil {
		t.Fatal(err)
	}

	// parsed, issues := env.Parse(`test.Add()==3 && test.Subtract()==-1 && MulBy3(9)==27`)
	parsed, issues := env.Parse(`test.a==3`)
	if issues != nil && issues.Err() != nil {
		log.Fatalf("parse error: %s", issues.Err())
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		log.Fatalf("type-check error: %s", issues.Err())

	}

	globalFunctions := cel.Functions(
		&functions.Overload{
			Operator: "MulBy3",
			Unary: func(lhs ref.Val) ref.Val {
				return types.Int(3 * lhs.Value().(int64))
			}})

	prg, err := env.Program(checked, globalFunctions)
	if err != nil {
		log.Fatalf("program construction error: %s", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{"test": Test{a: 1, b: 2}})
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(out)
	}
}
