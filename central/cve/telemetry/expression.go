package telemetry

import (
	"math"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/glob"
)

type expression struct {
	label Label
	op    string
	arg   string
}

func (e *expression) String() string {
	return string(e.label) + e.op + e.arg
}

func (e expression) Equal(s any) bool {
	return e.String() == s
}

// match returns whether the labels match the expression and the matched label
// value, if matched.
func (e *expression) match(labels func(Label) string) (string, bool) {
	if e == nil {
		return "", true
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
