package pgsearch

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
		validPrefixes = append(validPrefixes, "==")
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

func createNumericQuery(root string, prefix string, value float64) WhereClause {
	var valueStr string
	if _, fraction := math.Modf(value); fraction > 0 {
		valueStr = fmt.Sprintf("%0.2f", value)
	} else {
		valueStr = fmt.Sprintf("%0.0f", value)
	}

	if prefix == "" || prefix == "==" {
		prefix = "="
	}
	return WhereClause{
		Query:            fmt.Sprintf("%s %s $$", root, prefix),
		Values:           []interface{}{valueStr},
		equivalentGoFunc: getEquivalentGoFuncForNumericQuery(prefix, value),
	}
}

func newNumericQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	if len(ctx.queryModifiers) > 0 {
		return nil, errors.Errorf("modifiers not supported for numeric query: %+v", ctx.queryModifiers)
	}
	prefix, trimmedValue := parseNumericPrefix(ctx.value)
	valuePtr, err := parseNumericStringToFloat(trimmedValue)
	if err != nil {
		return nil, err
	}
	whereClause := createNumericQuery(ctx.qualifiedColumnName, prefix, valuePtr)
	return qeWithSelectFieldIfNeeded(ctx, &whereClause, nil), nil
}
