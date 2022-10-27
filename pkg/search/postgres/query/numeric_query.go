package pgsearch

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/readable"
)

type prefixAndInversion struct {
	prefix    string
	inversion string
}

var (
	prefixesAndInversions = []prefixAndInversion{
		{"<", ">="},
		{">", "<="},
	}

	validPrefixesSortedByLengthDec = func() []string {
		var validPrefixes []string
		for _, pAndI := range prefixesAndInversions {
			validPrefixes = append(validPrefixes, pAndI.prefix)
			validPrefixes = append(validPrefixes, pAndI.inversion)
		}
		validPrefixes = append(validPrefixes, "=", "==")
		sort.Slice(validPrefixes, func(i, j int) bool {
			return len(validPrefixes[i]) > len(validPrefixes[j])
		})
		return validPrefixes
	}()

	prefixesToInversions = func() map[string]string {
		out := make(map[string]string)
		for _, pAndI := range prefixesAndInversions {
			out[pAndI.prefix] = pAndI.inversion
			out[pAndI.inversion] = pAndI.prefix
		}
		return out
	}()
)

func parseNumericPrefix(value string) (prefix string, trimmedValue string) {
	for _, prefix := range validPrefixesSortedByLengthDec {
		if strings.HasPrefix(value, prefix) {
			return prefix, strings.TrimSpace(strings.TrimPrefix(value, prefix))
		}
	}
	return "", value
}

func parseNumericStringToFloat(s string) (float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func invertNumericPrefix(prefix string) string {
	return prefixesToInversions[prefix]
}

func getValueAsFloat64(foundValue interface{}) (float64, bool) {
	switch foundValue := foundValue.(type) {
	case float64:
		return foundValue, true
	case int:
		return float64(foundValue), true
	case int64:
		return float64(foundValue), true
	}
	return 0, false
}

func getComparator(prefix string) func(a, b float64) bool {
	switch prefix {
	case "<":
		return func(a, b float64) bool {
			return a < b
		}
	case ">":
		return func(a, b float64) bool {
			return a > b
		}
	case "<=":
		return func(a, b float64) bool {
			return a <= b
		}
	case ">=":
		return func(a, b float64) bool {
			return a >= b
		}
	}
	return func(a, b float64) bool {
		return a == b
	}
}

func getEquivalentGoFuncForNumericQuery(prefix string, value float64) func(foundValue interface{}) bool {
	comparator := getComparator(prefix)
	return func(foundValue interface{}) bool {
		asFloat, ok := getValueAsFloat64(foundValue)
		if !ok {
			return false
		}
		return comparator(asFloat, value)
	}
}

func createNumericRangeQuery(root string, lower float64, upper float64) WhereClause {
	lowerStr := readable.Float(lower, 2)
	upperStr := readable.Float(upper, 2)
	return WhereClause{
		Query:  fmt.Sprintf("(%s > $$) AND (%s < $$)", root, root),
		Values: []interface{}{lowerStr, upperStr},
		equivalentGoFunc: func(foundValue interface{}) bool {
			asFloat, ok := getValueAsFloat64(foundValue)
			if !ok {
				return false
			}
			return asFloat > lower && asFloat < upper
		},
	}
}

func createNumericPrefixQuery(root string, prefix string, value float64) WhereClause {
	valueStr := readable.Float(value, 2)

	if prefix == "" || prefix == "==" {
		prefix = "="
	}
	return WhereClause{
		Query:            fmt.Sprintf("%s %s $$", root, prefix),
		Values:           []interface{}{valueStr},
		equivalentGoFunc: getEquivalentGoFuncForNumericQuery(prefix, value),
	}
}

var (
	errNotARange = errors.New("not a range")
)

func maybeParseNumericRange(value string) (float64, float64, error) {
	// Split the value into two parts, separated by a hyphen.
	// We need to be careful to ensure that we don't mistake
	// hyphens for minus signs.
	for i, char := range value {
		if i == 0 {
			continue
		}
		if char == '-' {
			lower, err := parseNumericStringToFloat(value[:i])
			if err != nil {
				return 0, 0, errors.Errorf("invalid range %s (%v)", value, err)
			}
			upper, err := parseNumericStringToFloat(value[i+1:])
			if err != nil {
				return 0, 0, errors.Errorf("invalid range %s (%v)", value, err)
			}
			if lower >= upper {
				return 0, 0, errors.Errorf("invalid range %s (first value must be strictly less than the second)", value)
			}
			return lower, upper, nil
		}
	}
	return 0, 0, errNotARange
}

func newNumericQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	if len(ctx.queryModifiers) > 0 {
		return nil, errors.Errorf("modifiers not supported for numeric query: %+v", ctx.queryModifiers)
	}
	prefix, trimmedValue := parseNumericPrefix(ctx.value)

	var whereClause WhereClause
	var isRange bool
	if prefix == "" {
		lower, upper, err := maybeParseNumericRange(trimmedValue)
		if err == nil {
			whereClause = createNumericRangeQuery(ctx.qualifiedColumnName, lower, upper)
			isRange = true
		} else if err != errNotARange {
			return nil, errors.Wrap(err, "range query was attempted, but it was invalid")
		}
	}
	if !isRange {
		floatValue, err := parseNumericStringToFloat(trimmedValue)
		if err != nil {
			return nil, err
		}
		whereClause = createNumericPrefixQuery(ctx.qualifiedColumnName, prefix, floatValue)
	}
	return qeWithSelectFieldIfNeeded(ctx, &whereClause, nil), nil
}
