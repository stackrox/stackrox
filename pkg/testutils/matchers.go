package testutils

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	testingTTy = reflect.TypeOf((*assert.TestingT)(nil)).Elem()
)

type predMatcher struct {
	desc    string
	inTy    reflect.Type
	checker reflect.Value
}

// PredMatcher returns a gomock matcher that applies the given checker (which must be a unary function with a bool return
// value) to its argument.
func PredMatcher(desc string, checker interface{}) gomock.Matcher {
	ty := reflect.TypeOf(checker)

	if ty.Kind() != reflect.Func {
		panic("predicate matcher requires a function argument")
	}

	if ty.NumIn() != 1 {
		panic("function for predicate matcher must have exactly one input parameter")
	}

	if ty.NumOut() != 1 {
		panic("function for predicate matcher must have exactly one output parameter")
	}

	outTy := ty.Out(0)
	if outTy.Kind() != reflect.Bool {
		panic("function for predicate matcher must have a boolean return value")
	}

	return predMatcher{
		desc:    desc,
		inTy:    ty.In(0),
		checker: reflect.ValueOf(checker),
	}
}

func (p predMatcher) String() string {
	return p.desc
}

func (p predMatcher) Matches(x interface{}) bool {
	v := reflect.ValueOf(x)
	if !v.Type().AssignableTo(p.inTy) {
		return false
	}
	out := p.checker.Call([]reflect.Value{v})
	return out[0].Bool()
}

type failureRecorder bool

func (r *failureRecorder) Errorf(_ string, _ ...interface{}) {
	*r = true
}

type assertionMatcher struct {
	assertFunc reflect.Value
	staticArgs []reflect.Value
}

// AssertionMatcher returns a matcher using a function from the `assert` package for checking.
func AssertionMatcher(assertFn interface{}, args ...interface{}) gomock.Matcher {
	assertFnVal := reflect.ValueOf(assertFn)
	if assertFnVal.Kind() != reflect.Func {
		panic("AssertionMatcher requires a function argument")
	}

	assertFnTy := assertFnVal.Type()

	expectedParamCount := 1 + len(args) + 1
	if assertFnTy.IsVariadic() {
		expectedParamCount++
	}

	if assertFnTy.NumIn() != expectedParamCount {
		panic("AssertionMatcher requires a function taking at least 2 arguments")
	}
	param0Ty := assertFnTy.In(0)
	if param0Ty != testingTTy {
		panic("AssertionMatcher requires a function taking a TestingT as its first parameter")
	}

	if assertFnTy.NumOut() != 1 {
		panic("AssertionMatcher requires a function returning exactly one value")
	}
	if assertFnTy.Out(0).Kind() != reflect.Bool {
		panic("AssertionMatcher requires a function returning a bool-like value")
	}

	argVals := make([]reflect.Value, len(args))
	for i, arg := range args {
		argVals[i] = reflect.ValueOf(arg)
	}

	return &assertionMatcher{
		assertFunc: assertFnVal,
		staticArgs: argVals,
	}
}

func (m *assertionMatcher) String() string {
	var argStrings []string
	for _, argVal := range m.staticArgs {
		argStrings = append(argStrings, fmt.Sprintf("%s", argVal.Interface()))
	}

	funcName := runtime.FuncForPC(m.assertFunc.Pointer()).Name()
	lastSlashIdx := strings.LastIndex(funcName, "/")
	if lastSlashIdx != -1 {
		funcName = funcName[lastSlashIdx+1:]
	}
	return fmt.Sprintf("%s(%s)", funcName, strings.Join(argStrings, ", "))
}

func (m *assertionMatcher) Matches(x interface{}) bool {
	var failed failureRecorder
	args := make([]reflect.Value, 0, len(m.staticArgs)+2)
	args = append(args, reflect.ValueOf(&failed))
	args = append(args, m.staticArgs...)
	args = append(args, reflect.ValueOf(x))

	outs := m.assertFunc.Call(args)
	return outs[0].Bool()
}
