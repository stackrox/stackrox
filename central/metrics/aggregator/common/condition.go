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

var knownOperators = []operator{opEQ, opNE, opGT, opGE, opLT, opLE, opOR}

type Condition struct {
	op  operator
	arg string
}

// MakeCondition constructs an condition.
func MakeCondition(op, arg string) (*Condition, error) {
	cond := &Condition{operator(op), arg}
	if err := cond.validate(); err != nil {
		return nil, err
	}
	return cond, nil
}

// MustMakeCondition constructs an condition and panics on error.
func MustMakeCondition(op, arg string) *Condition {
	cond, err := MakeCondition(op, arg)
	if err != nil {
		panic(err)
	}
	return cond
}

func (c *Condition) validate() error {
	switch {
	// Test operator:
	case c.op == opZ:
		if len(c.arg) > 0 {
			return fmt.Errorf("missing operator in %q", c)
		}
		return errors.New("empty operator")
	case !slices.Contains(knownOperators, c.op):
		return fmt.Errorf("operator in %q is not one of %q", c, knownOperators)
	// Test argument:
	case c.op == opOR:
		if len(c.arg) > 0 {
			return fmt.Errorf("unexpected argument in %q", c)
		}
	case len(c.arg) == 0:
		return fmt.Errorf("missing argument in %q", c)
	case !c.isFloatArg() && !c.isGlobArg():
		return fmt.Errorf("cannot parse the argument in %q", c)
	}
	return nil
}

func (c *Condition) Equals(b *Condition) bool {
	return c == b || c != nil && b != nil && c.op == b.op && c.arg == b.arg
}

func (c *Condition) String() string {
	return string(c.op) + c.arg
}

func (c *Condition) isFloatArg() bool {
	_, err := strconv.ParseFloat(c.arg, 32)
	return err == nil
}

func (c *Condition) isGlobArg() bool {
	return glob.Pattern(c.arg).Ptr().Compile() == nil
}

// match returns whether the labels match the condition and the matched label
// value, if matched.
func (c *Condition) match(value string) bool {
	if c == nil {
		return true
	}
	if argument, err := strconv.ParseFloat(c.arg, 32); err == nil {
		if numericValue, err := strconv.ParseFloat(value, 32); err == nil {
			return c.compareFloats(numericValue, argument)
		}
	}
	return c.compareStrings(value, c.arg)
}

func (c *Condition) compareStrings(a, b string) bool {
	switch c.op {
	case "":
		return a != "" && b == ""
	case opEQ:
		return glob.Pattern(b).Ptr().Match(a)
	case opNE:
		return !glob.Pattern(b).Ptr().Match(a)
	case opGT:
		return strings.Compare(strings.ToLower(a), strings.ToLower(b)) > 0
	case opGE:
		return strings.EqualFold(a, b) ||
			strings.Compare(strings.ToLower(a), strings.ToLower(b)) > 0
	case opLT:
		return strings.Compare(strings.ToLower(a), strings.ToLower(b)) < 0
	case opLE:
		return strings.EqualFold(a, b) ||
			strings.Compare(strings.ToLower(a), strings.ToLower(b)) < 0
	}
	return false
}

func (c *Condition) compareFloats(a, b float64) bool {
	const epsilon = 1e-9
	switch c.op {
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
