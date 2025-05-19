package common

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/glob"
)

type operator string

const (
	opZ  operator = ""
	opEQ operator = "="
	opNE operator = "!="
	opGT operator = ">"
	opGE operator = ">="
	opLT operator = "<"
	opLE operator = "<="

	opOR operator = "OR"
)

type Expression struct {
	op  operator
	arg string
}

// MakeExpression constructs an expression.
func MakeExpression(op, arg string) (*Expression, error) {
	expr := &Expression{operator(op), arg}
	if err := expr.validate(); err != nil {
		return nil, err
	}
	return expr, nil
}

// MustMakeExpression constructs an expression and panics on error.
func MustMakeExpression(op, arg string) *Expression {
	expr, err := MakeExpression(op, arg)
	if err != nil {
		panic(err)
	}
	return expr
}

func (e *Expression) validate() error {
	switch {
	// Test operator:
	case e.op == opZ:
		if len(e.arg) > 0 {
			return fmt.Errorf("missing operator in %q", e)
		}
		return errors.New("empty operator")
	case !slices.Contains([]operator{opEQ, opNE, opGT, opGE, opLT, opLE, opOR}, e.op):
		return fmt.Errorf("unknown operator in %q", e)
	// Test argument:
	case e.op == opOR:
		if len(e.arg) > 0 {
			return fmt.Errorf("unexpected argument in %q", e)
		}
	case len(e.arg) == 0:
		return fmt.Errorf("missing argument in %q", e)
	case !e.isFloatArg() && !e.isGlobArg():
		return fmt.Errorf("cannot parse the argument in %q", e)
	}
	return nil
}

func (e *Expression) String() string {
	return string(e.op) + e.arg
}

func (e *Expression) isFloatArg() bool {
	_, err := strconv.ParseFloat(e.arg, 32)
	return err == nil
}

func (e *Expression) isGlobArg() bool {
	return glob.Pattern(e.arg).Ptr().Compile() == nil
}

// match returns whether the labels match the expression and the matched label
// value, if matched.
func (e *Expression) match(value string) bool {
	if e == nil {
		return true
	}
	if argument, err := strconv.ParseFloat(e.arg, 32); err == nil {
		if numericValue, err := strconv.ParseFloat(value, 32); err == nil {
			return e.compareFloats(numericValue, argument)
		}
	}
	return e.compareStrings(value, e.arg)
}

func (e *Expression) compareStrings(a, b string) bool {
	switch e.op {
	case "":
		return a != "" && b == ""
	case opEQ:
		return glob.Pattern(b).Ptr().Match(a)
	case opNE:
		return !glob.Pattern(b).Ptr().Match(a)
	case opGT:
		return strings.Compare(a, b) > 0
	case opGE:
		return strings.EqualFold(a, b) ||
			strings.Compare(a, b) > 0
	case opLT:
		return strings.Compare(a, b) < 0
	case opLE:
		return strings.EqualFold(a, b) ||
			strings.Compare(a, b) < 0
	}
	return false
}

func (e *Expression) compareFloats(a, b float64) bool {
	const epsilon = 1e-9
	switch e.op {
	case opEQ:
		return math.Abs(a-b) <= epsilon
	case opNE:
		return math.Abs(a-b) > epsilon
	case opGT:
		return a > b
	case opGE:
		return a >= b
	case opLT:
		return a < b
	case opLE:
		return a <= b
	}
	return false
}
