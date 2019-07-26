package blevesearch

import (
	"fmt"
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

	switch prefix {
	case "<=":
		maxInclusive = boolPtr(true)
		max = value
	case "<":
		maxInclusive = boolPtr(false)
		max = value
	case ">=":
		minInclusive = boolPtr(true)
		min = value
	case ">":
		minInclusive = boolPtr(false)
		min = value
	default:
		minInclusive = boolPtr(true)
		maxInclusive = boolPtr(true)
		min = value
		max = value
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
