package blevesearch

import (
	"fmt"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeFieldName = "blah"
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
			expected: getExpectedNumericQuery(nil, floatPtr(num1-delta), nil, boolPtr(false)),
		},
		{
			name:     fmt.Sprintf("less than or equals to %v", num1),
			value:    floatPtr(num1),
			prefix:   "<=",
			expected: getExpectedNumericQuery(nil, floatPtr(num1+delta), nil, boolPtr(true)),
		},
		{
			name:     fmt.Sprintf("equals to %v", num1),
			value:    floatPtr(num1),
			prefix:   "=",
			expected: getExpectedNumericQuery(floatPtr(num1-delta), floatPtr(num1+delta), boolPtr(true), boolPtr(true)),
		},
		{
			name:     fmt.Sprintf("equals to %v", num2),
			value:    floatPtr(num2),
			prefix:   "=",
			expected: getExpectedNumericQuery(floatPtr(num2-delta), floatPtr(num2+delta), boolPtr(true), boolPtr(true)),
		},
		{
			name:     fmt.Sprintf("greater than or equals to %v", num1),
			value:    floatPtr(num1),
			prefix:   ">=",
			expected: getExpectedNumericQuery(floatPtr(num1-delta), nil, boolPtr(true), nil),
		},
		{
			name:     fmt.Sprintf("greater than to %v", num2),
			value:    floatPtr(num2),
			prefix:   ">",
			expected: getExpectedNumericQuery(floatPtr(float64(num2+delta)), nil, boolPtr(false), nil),
		},
		{
			name:     "no value",
			value:    nil,
			prefix:   ">=",
			expected: getExpectedNumericQuery(nil, nil, boolPtr(true), nil),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := createNumericQuery(fakeFieldName, c.prefix, c.value)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestCreateNumericQuery(t *testing.T) {
	cases := []struct {
		value     string
		expectErr bool
		expected  query.Query
	}{
		{
			value:    "-1",
			expected: getExpectedNumericQuery(floatPtr(-1), floatPtr(-1), boolPtr(true), boolPtr(true)),
		},
		{
			value:     "-1--2",
			expectErr: true,
		},
		{
			value:    "-2--1",
			expected: getExpectedNumericQuery(floatPtr(-2), floatPtr(-1), boolPtr(false), boolPtr(false)),
		},
		{
			value:    "-2-1",
			expected: getExpectedNumericQuery(floatPtr(-2), floatPtr(1), boolPtr(false), boolPtr(false)),
		},
		{
			value:     "1-1",
			expectErr: true,
		},
		{
			value:     "2-1",
			expectErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			actual, err := newNumericQuery(0, "blah", c.value)
			if c.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func getExpectedNumericQuery(min, max *float64, minInclusive, maxInclusive *bool) query.Query {
	q := bleve.NewNumericRangeInclusiveQuery(min, max, minInclusive, maxInclusive)
	q.SetField(fakeFieldName)
	return q
}
