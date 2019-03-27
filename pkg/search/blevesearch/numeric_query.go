package blevesearch

import (
	"strconv"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

func parseNumericPrefix(value string) (prefix string, trimmedValue string) {
	for _, prefix := range []string{"<=", ">=", "<", ">"} {
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

func newNumericQuery(_ v1.SearchCategory, field string, value string) (query.Query, error) {
	prefix, trimmedValue := parseNumericPrefix(value)
	valuePtr, err := parseNumericStringToPtr(trimmedValue)
	if err != nil {
		return nil, err
	}
	return createNumericQuery(field, prefix, valuePtr), nil
}
