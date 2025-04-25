package telemetry

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/glob"
)

var ops = []string{"!=", "=", "<=", ">=", "<", ">"} // The order matters!

// getLabel returns the labels key from the provided expression.
//
// Example:
//
//	getLabel("a=b") == "a"
func getLabel(expr expression) string {
	if i := strings.IndexFunc(string(expr), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.')
	}); i > 0 {
		return string(expr)[:i]
	}
	return string(expr)
}

// splitExpression splits an expression to the label, operator and argument.
//
// Example:
//
//	"a=b": ("a", "=", "b")
func splitExpression(expr expression) (string, string, string) {
	label := getLabel(expr)
	if len(expr) == len(label) {
		return label, "", ""
	}
	opArg := string(expr[len(label):])
	for _, op := range ops {
		if strings.HasPrefix(opArg, op) {
			return label, op, opArg[len(op):]
		}
	}
	return label, "", opArg
}

func filter(expr expression, labels func(string) string) (string, string, bool) {
	label, op, arg := splitExpression(expr)
	value := labels(label)
	var err error
	switch op {
	case "!=":
		fallthrough
	case "=":
		pattern := glob.Pattern(arg)
		return label, value, op == "=" && pattern.Match(value) ||
			op == "!=" && !pattern.Match(value)
	case ">":
		fallthrough
	case ">=":
		fallthrough
	case "<":
		fallthrough
	case "<=":
		argument := 0.0
		argument, err = strconv.ParseFloat(arg, 32)
		if err != nil {
			return label, value, false
		}
		number, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return label, value, false
		}
		switch op {
		case ">":
			return label, value, number > argument
		case ">=":
			return label, value, number >= argument
		case "<":
			return label, value, number < argument
		case "<=":
			return label, value, number <= argument
		}
	}
	return label, value, value != ""
}
