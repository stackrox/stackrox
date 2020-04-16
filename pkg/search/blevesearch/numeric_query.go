package blevesearch

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type prefixAndInversion struct {
	prefix    string
	inversion string
}

const (
	// We use delta because of precision difference between float32 and float64.
	// For example, 4.6 -> 4.599999904632568, 9.8 -> 9.800000190734863
	numericQueryDelta = 0.01
)

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

// NumericQueryValue represents a parsed numeric query.
type NumericQueryValue struct {
	Comparator storage.Comparator
	Value      float64
}

// ParseNumericQueryValue interprets a string a NumericQueryValue.
func ParseNumericQueryValue(value string) (NumericQueryValue, error) {
	prefix, trimmedValue := parseNumericPrefix(value)
	valuePtr, err := parseNumericStringToPtr(trimmedValue)
	if err != nil {
		return NumericQueryValue{}, err
	}

	output := NumericQueryValue{
		Value: *valuePtr,
	}
	switch prefix {
	case "<=":
		output.Comparator = storage.Comparator_LESS_THAN_OR_EQUALS
	case "<":
		output.Comparator = storage.Comparator_LESS_THAN
	case ">=":
		output.Comparator = storage.Comparator_GREATER_THAN_OR_EQUALS
	case ">":
		output.Comparator = storage.Comparator_GREATER_THAN
	case "==":
		output.Comparator = storage.Comparator_EQUALS
	default:
		return NumericQueryValue{}, fmt.Errorf("unrecognized comparator in query %s", prefix)
	}
	return output, nil
}

// PrintNumericQueryValue prints out the input NumericQueryValue as a query string.
func PrintNumericQueryValue(value NumericQueryValue) string {
	var prefix string
	switch value.Comparator {
	case storage.Comparator_LESS_THAN_OR_EQUALS:
		prefix = "<="
	case storage.Comparator_LESS_THAN:
		prefix = "<"
	case storage.Comparator_GREATER_THAN_OR_EQUALS:
		prefix = ">="
	case storage.Comparator_GREATER_THAN:
		prefix = ">"
	case storage.Comparator_EQUALS:
		prefix = "=="
	}
	return fmt.Sprintf("%s%f", prefix, value.Value)
}

func invertNumericPrefix(prefix string) string {
	return prefixesToInversions[prefix]
}

func parseNumericPrefix(value string) (prefix string, trimmedValue string) {
	for _, prefix := range validPrefixesSortedByLengthDec {
		if strings.HasPrefix(value, prefix) {
			return prefix, strings.TrimPrefix(value, prefix)
		}
	}
	return "", value
}

func boolPtr(b bool) *bool {
	return &b
}

func floatPtr(f float64) *float64 {
	return &f
}

func parseNumericStringToPtr(s string) (*float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func createNumericQuery(field string, prefix string, value *float64) query.Query {
	var min, max *float64
	var maxInclusive, minInclusive *bool

	var delta float64
	// For performance reasons, if there is no fractional part of the float64 then
	// simply use the passed float64
	if value != nil {
		if _, fraction := math.Modf(*value); fraction > 0 {
			delta = numericQueryDelta
		}
	}

	switch prefix {
	case "<=":
		maxInclusive = boolPtr(true)
		// adjust to include in cases such as 9.8 -> 9.800000190734863
		max = adjustValue(value, delta)
	case "<":
		maxInclusive = boolPtr(false)
		// adjust to exclude in cases such as 4.6 -> 4.599999904632568
		max = adjustValue(value, -delta)
	case ">=":
		minInclusive = boolPtr(true)
		// adjust to include in cases such as 4.6 -> 4.599999904632568
		min = adjustValue(value, -delta)
	case ">":
		minInclusive = boolPtr(false)
		// adjust to exclude in cases such as 9.8 -> 9.800000190734863
		min = adjustValue(value, delta)
	default:
		minInclusive = boolPtr(true)
		maxInclusive = boolPtr(true)

		if value != nil {
			min = adjustValue(value, -delta)
			max = adjustValue(value, delta)
		}
	}
	q := bleve.NewNumericRangeInclusiveQuery(min, max, minInclusive, maxInclusive)
	q.SetField(field)
	return q
}

func newNumericQuery(_ v1.SearchCategory, field string, value string, modifiers ...queryModifier) (query.Query, error) {
	if len(modifiers) > 0 {
		return nil, errors.Errorf("modifiers not supported for numeric query: %+v", modifiers)
	}
	prefix, trimmedValue := parseNumericPrefix(value)
	valuePtr, err := parseNumericStringToPtr(trimmedValue)
	if err != nil {
		return nil, err
	}
	return createNumericQuery(field, prefix, valuePtr), nil
}

func adjustValue(val *float64, delta float64) *float64 {
	if val == nil {
		return nil
	}
	return floatPtr(*val + delta)
}
