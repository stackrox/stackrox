package basematchers

import (
	"fmt"
	"math"
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

const (
	floatTol = 1e-4
)

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
		return func(a, b float64) bool { return math.Abs(a-b) < floatTol }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

// ParseNumericPrefix splits a numeric comparison such as ">= 75" into its operator
// and the remainder. Two-character operators are checked first so "<=" is not read as "<".
func ParseNumericPrefix(value string) (prefix string, trimmedValue string) {
	for _, prefix := range []string{LessThanOrEqualTo, GreaterThanOrEqualTo, LessThan, GreaterThan} {
		if strings.HasPrefix(value, prefix) {
			return prefix, strings.TrimSpace(value[len(prefix):])
		}
	}
	return "", value
}
