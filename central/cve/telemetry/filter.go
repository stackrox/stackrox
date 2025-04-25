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
func getLabel(expr string) string {
	if i := strings.IndexFunc(expr, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.')
	}); i > 0 {
		return expr[:i]
	}
	return expr
}

// makeExpression splits an expression to the label, operator and argument.
//
// Example:
//
//	"a=b": ("a", "=", "b")
func makeExpression(expr string) expression {
	label := getLabel(expr)
	if len(expr) == len(label) {
		return expression{label, "", ""}
	}
	opArg := string(expr[len(label):])
	for _, op := range ops {
		if strings.HasPrefix(opArg, op) {
			return expression{label, op, opArg[len(op):]}
		}
	}
	return expression{label, "", opArg}
}

func filter(expr expression, labels func(string) string) (string, bool) {
	value := labels(expr.label)
	var err error
	switch expr.op {
	case "":
		return value, value != "" && expr.arg == ""
	case "!=":
		fallthrough
	case "=":
		pattern := glob.Pattern(expr.arg)
		return value, expr.op == "=" && pattern.Match(value) ||
			expr.op == "!=" && !pattern.Match(value)
	case ">":
		fallthrough
	case ">=":
		fallthrough
	case "<":
		fallthrough
	case "<=":
		argument := 0.0
		argument, err = strconv.ParseFloat(expr.arg, 32)
		if err != nil {
			return value, false
		}
		number, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return value, false
		}
		switch expr.op {
		case ">":
			return value, number > argument
		case ">=":
			return value, number >= argument
		case "<":
			return value, number < argument
		case "<=":
			return value, number <= argument
		}
	}
	return value, value != ""
}
