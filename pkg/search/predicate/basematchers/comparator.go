package basematchers

import (
	"fmt"
	"strings"
)

// The following block enumerates numeric comparator prefixes.
const (
	LessThanOrEqualTo    = "<="
	GreaterThanOrEqualTo = ">="
	LessThan             = "<"
	GreaterThan          = ">"
)

func intComparator(cmp string) (func(a, b int64) bool, error) {
	switch cmp {
	case LessThanOrEqualTo:
		return func(a, b int64) bool { return a <= b }, nil
	case GreaterThanOrEqualTo:
		return func(a, b int64) bool { return a >= b }, nil
	case LessThan:
		return func(a, b int64) bool { return a < b }, nil
	case GreaterThan:
		return func(a, b int64) bool { return a > b }, nil
	case "":
		return func(a, b int64) bool { return a == b }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func uintComparator(cmp string) (func(a, b uint64) bool, error) {
	switch cmp {
	case LessThanOrEqualTo:
		return func(a, b uint64) bool { return a <= b }, nil
	case GreaterThanOrEqualTo:
		return func(a, b uint64) bool { return a >= b }, nil
	case LessThan:
		return func(a, b uint64) bool { return a < b }, nil
	case GreaterThan:
		return func(a, b uint64) bool { return a > b }, nil
	case "":
		return func(a, b uint64) bool { return a == b }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func floatComparator(cmp string) (func(a, b float64) bool, error) {
	switch cmp {
	case LessThanOrEqualTo:
		return func(a, b float64) bool { return a <= b }, nil
	case GreaterThanOrEqualTo:
		return func(a, b float64) bool { return a >= b }, nil
	case LessThan:
		return func(a, b float64) bool { return a < b }, nil
	case GreaterThan:
		return func(a, b float64) bool { return a > b }, nil
	case "":
		return func(a, b float64) bool { return a == b }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func parseNumericPrefix(value string) (prefix string, trimmedValue string) {
	// The order which these checks are executed must be maintained.
	// If we for instance look for "<" before "<=", we will never find "<=" because "<" will be found as its prefix.
	for _, prefix := range []string{LessThanOrEqualTo, GreaterThanOrEqualTo, LessThan, GreaterThan} {
		if strings.HasPrefix(value, prefix) {
			return prefix, strings.TrimSpace(value[len(prefix):])
		}
	}
	return "", value
}
