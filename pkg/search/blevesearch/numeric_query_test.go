package blevesearch

import (
	"fmt"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stretchr/testify/assert"
)

func TestParseNumericPrefix(t *testing.T) {
	var cases = []struct {
		value          string
		expectedPrefix string
		expectedValue  string
	}{
		{
			value:          "<=lol",
			expectedPrefix: "<=",
			expectedValue:  "lol",
		},
		{
			value:          ">lol",
			expectedPrefix: ">",
			expectedValue:  "lol",
		},
		{
			value:          ">=lol",
			expectedPrefix: ">=",
			expectedValue:  "lol",
		},
		{
			value:          ">lol",
			expectedPrefix: ">",
			expectedValue:  "lol",
		},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			prefix, value := parseNumericPrefix(c.value)
			assert.Equal(t, c.expectedPrefix, prefix)
			assert.Equal(t, c.expectedValue, value)
		})
	}
}

func TestNumberDelta(t *testing.T) {
	num1 := float64(4.6)
	num2 := float64(9.8)
	delta := float64(0.01)
	var cases = []struct {
		name     string
		value    *float64
		prefix   string
		expected query.Query
	}{
		{
			name:     fmt.Sprintf("less than %v", num1),
			value:    floatPtr(num1),
			prefix:   "<",
			expected: getExpectedNumericQuery("blah", nil, floatPtr(num1-delta), nil, boolPtr(false)),
		},
		{
			name:     fmt.Sprintf("less than or equals to %v", num1),
			value:    floatPtr(num1),
			prefix:   "<=",
			expected: getExpectedNumericQuery("blah", nil, floatPtr(num1+delta), nil, boolPtr(true)),
		},
		{
			name:     fmt.Sprintf("equals to %v", num1),
			value:    floatPtr(num1),
			prefix:   "=",
			expected: getExpectedNumericQuery("blah", floatPtr(num1-delta), floatPtr(num1+delta), boolPtr(true), boolPtr(true)),
		},
		{
			name:     fmt.Sprintf("equals to %v", num2),
			value:    floatPtr(num2),
			prefix:   "=",
			expected: getExpectedNumericQuery("blah", floatPtr(num2-delta), floatPtr(num2+delta), boolPtr(true), boolPtr(true)),
		},
		{
			name:     fmt.Sprintf("greater than or equals to %v", num1),
			value:    floatPtr(num1),
			prefix:   ">=",
			expected: getExpectedNumericQuery("blah", floatPtr(num1-delta), nil, boolPtr(true), nil),
		},
		{
			name:     fmt.Sprintf("greater than to %v", num2),
			value:    floatPtr(num2),
			prefix:   ">",
			expected: getExpectedNumericQuery("blah", floatPtr(float64(num2+delta)), nil, boolPtr(false), nil),
		},
		{
			name:     "no value",
			value:    nil,
			prefix:   ">=",
			expected: getExpectedNumericQuery("blah", nil, nil, boolPtr(true), nil),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := createNumericQuery("blah", c.prefix, c.value)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func getExpectedNumericQuery(field string, min, max *float64, minInclusive, maxInclusive *bool) query.Query {
	q := bleve.NewNumericRangeInclusiveQuery(min, max, minInclusive, maxInclusive)
	q.SetField(field)
	return q
}
