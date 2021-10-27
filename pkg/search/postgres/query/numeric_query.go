package pgsearch

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
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

func createNumericQuery(table string, field *search.Field, prefix string, value *float64) *QueryEntry {
	var min, max *float64
	var delta float64
	// For performance reasons, if there is no fractional part of the float64 then
	// simply use the passed float64
	if value != nil {
		if _, fraction := math.Modf(*value); fraction > 0 {
			delta = numericQueryDelta
		}
	}

	var equality bool
	switch prefix {
	case "<=":
		// adjust to include in cases such as 9.8 -> 9.800000190734863
		value = adjustValue(value, delta)
	case "<":
		value = adjustValue(value, -delta)
	case ">=":
		// adjust to include in cases such as 4.6 -> 4.599999904632568
		value = adjustValue(value, -delta)
	case ">":
		// adjust to exclude in cases such as 9.8 -> 9.800000190734863
		value = adjustValue(value, delta)
	default:
		equality = true
		if value != nil && delta != 0 {
			min = adjustValue(value, -delta)
			max = adjustValue(value, delta)
		}
	}

	elemPath := GenerateShortestElemPath(table, field.Elems)

	root := field.TopLevelValue()
	if root == "" {
		root = fmt.Sprintf("(%s)::numeric", RenderFinalPath(elemPath, field.LastElem().ProtoJSONName))
	}
	if equality {
		if delta == 0 {
			// min and max are the same
			return &QueryEntry{
				Query:  fmt.Sprintf("%s = $$", root),
				Values: []interface{}{*value},
			}
		}
		return &QueryEntry{
			Query:  fmt.Sprintf("%s >= $$ and %s <= $$", root, root),
			Values: []interface{}{*min, *max},
		}
	}
	var val float64
	if min != nil {
		val = *min
	} else {
		val = *max
	}
	return &QueryEntry{
		Query:  fmt.Sprintf("%s %s $$", root, prefix),
		Values: []interface{}{val},
	}
}

func newNumericQuery(table string, field *search.Field, value string, modifiers ...search.QueryModifier) (*QueryEntry, error) {
	if len(modifiers) > 0 {
		return nil, errors.Errorf("modifiers not supported for numeric query: %+v", modifiers)
	}
	prefix, trimmedValue := parseNumericPrefix(value)
	valuePtr, err := parseNumericStringToPtr(trimmedValue)
	if err != nil {
		return nil, err
	}
	return createNumericQuery(table, field, prefix, valuePtr), nil
}

func adjustValue(val *float64, delta float64) *float64 {
	if val == nil {
		return nil
	}
	return floatPtr(*val + delta)
}
