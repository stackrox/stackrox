package telemetry

import (
	"math"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/glob"
)

var ops = []string{"!=", "=", "<=", ">=", "<", ">"} // The order matters!

type expression struct {
	label Label
	op    string
	arg   string
}

// getLabel returns the labels key from the provided expression.
//
// Example:
//
//	getLabel("a=b") == "a"
func getLabel(expr string) Label {
	if i := strings.IndexFunc(expr, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.')
	}); i >= 0 {
		return Label(expr[:i])
	}
	return Label(expr)
}

// makeExpression splits an expression to the label, operator and argument.
//
// Example:
//
//	"a=b": ("a", "=", "b")
func makeExpression(expr string) *expression {
	expr = strings.Trim(expr, " ")
	label := getLabel(expr)
	if len(label) == 0 {
		return nil
	}
	if len(expr) == len(label) {
		return &expression{label, "", ""}
	}
	opArg := strings.Trim(string(expr[len(label):]), " ")
	for _, op := range ops {
		if strings.HasPrefix(opArg, op) {
			arg := strings.Trim(opArg[len(op):], " ")
			if len(arg) == 0 {
				return nil
			}
			return &expression{label, op, arg}
		}
	}
	return nil
}

func (e *expression) String() string {
	return string(e.label) + e.op + e.arg
}

// match returns whether the labels match the expression and the matched label
// value, if matched.
func (e *expression) match(labels func(Label) string) (string, bool) {
	if e == nil {
		return "", false
	}
	value := labels(e.label)
	if argument, err := strconv.ParseFloat(e.arg, 32); err == nil {
		if numericValue, err := strconv.ParseFloat(value, 32); err == nil {
			return value, e.compareFloats(numericValue, argument)
		}
	}
	return value, e.compareStrings(value, e.arg)
}

func (e *expression) compareStrings(a, b string) bool {
	switch e.op {
	case "":
		return a != "" && b == ""
	case "=":
		return glob.Pattern(b).Ptr().Match(a)
	case "!=":
		return !glob.Pattern(b).Ptr().Match(a)
	case ">":
		return strings.Compare(a, b) > 0
	case ">=":
		return strings.EqualFold(a, b) ||
			strings.Compare(a, b) > 0
	case "<":
		return strings.Compare(a, b) < 0
	case "<=":
		return strings.EqualFold(a, b) ||
			strings.Compare(a, b) < 0
	}
	return false
}

func (e *expression) compareFloats(a, b float64) bool {
	const epsilon = 1e-9
	switch e.op {
	case "=":
		return math.Abs(a-b) <= epsilon
	case "!=":
		return math.Abs(a-b) > epsilon
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}
