package blevesearch

import (
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
