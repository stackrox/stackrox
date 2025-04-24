package telemetry

import (
	"strconv"
	"strings"

	"github.com/gobwas/glob"
)

var ops = []string{"!=", "=", "<=", ">=", "<", ">"} // The order matters!

var globCache map[string]glob.Glob

func getKey(key string) string {
	for i, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.') {
			return key[:i]
		}
	}
	return key
}

func splitExpression(s string) (string, string, string) {
	key := getKey(s)
	if len(s) == len(key) {
		return key, "", ""
	}
	opExpr := s[len(key):]
	for _, op := range ops {
		if strings.HasPrefix(opExpr, op) {
			return key, op, opExpr[len(op):]
		}
	}
	return key, "", opExpr
}

func filter(expr expression, metric map[keyInstance]string) (string, bool) {
	key, op, arg := splitExpression(expr)
	var err error
	switch op {
	case "!=":
		fallthrough
	case "=":
		e := globCache[arg]
		if e == nil {
			if e, err = glob.Compile(arg); err != nil {
				return key, false
			}
			globCache[arg] = e
		}
		return key, op == "=" && e.Match(metric[key]) || op == "!=" && !e.Match(metric[key])
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
			return key, false
		}
		value, err := strconv.ParseFloat(metric[key], 32)
		if err != nil {
			return key, false
		}
		switch op {
		case ">":
			return key, value > argument
		case ">=":
			return key, value >= argument
		case "<":
			return key, value < argument
		case "<=":
			return key, value <= argument
		}
	}
	return key, metric[key] != ""
}
