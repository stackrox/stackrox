package predicate

import (
	"fmt"
	"strings"
)

// Produce a predicate for the given numerical or date time query.
var (
	lessThanOrEqualTo    = "<="
	greaterThanOrEqualTo = ">="
	lessThan             = "<"
	greaterThan          = ">"
)

func intComparator(cmp string) (func(a, b int64) bool, error) {
	switch cmp {
	case lessThanOrEqualTo:
		return func(a, b int64) bool { return a <= b }, nil
	case greaterThanOrEqualTo:
		return func(a, b int64) bool { return a >= b }, nil
	case lessThan:
		return func(a, b int64) bool { return a < b }, nil
	case greaterThan:
		return func(a, b int64) bool { return a > b }, nil
	case "":
		return func(a, b int64) bool { return a == b }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func uintComparator(cmp string) (func(a, b uint64) bool, error) {
	switch cmp {
	case lessThanOrEqualTo:
		return func(a, b uint64) bool { return a <= b }, nil
	case greaterThanOrEqualTo:
		return func(a, b uint64) bool { return a >= b }, nil
	case lessThan:
		return func(a, b uint64) bool { return a < b }, nil
	case greaterThan:
		return func(a, b uint64) bool { return a > b }, nil
	case "":
		return func(a, b uint64) bool { return a == b }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func floatComparator(cmp string) (func(a, b float64) bool, error) {
	switch cmp {
	case lessThanOrEqualTo:
		return func(a, b float64) bool { return a <= b }, nil
	case greaterThanOrEqualTo:
		return func(a, b float64) bool { return a >= b }, nil
	case lessThan:
		return func(a, b float64) bool { return a < b }, nil
	case greaterThan:
		return func(a, b float64) bool { return a > b }, nil
	case "":
		return func(a, b float64) bool { return a == b }, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func getNumericComparator(value string) (comparator string, string string) {
	// The order which these checks are executed must be maintained.
	// If we for instance look for "<" before "<=", we will never find "<=" because "<" will be found as it's prefix.
	if strings.HasPrefix(value, lessThanOrEqualTo) {
		return lessThanOrEqualTo, strings.TrimPrefix(value, lessThanOrEqualTo)
	} else if strings.HasPrefix(value, greaterThanOrEqualTo) {
		return greaterThanOrEqualTo, strings.TrimPrefix(value, greaterThanOrEqualTo)
	} else if strings.HasPrefix(value, lessThan) {
		return lessThan, strings.TrimPrefix(value, lessThan)
	} else if strings.HasPrefix(value, greaterThan) {
		return greaterThan, strings.TrimPrefix(value, greaterThan)
	}
	return "", value
}
