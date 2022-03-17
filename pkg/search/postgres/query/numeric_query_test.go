package pgsearch

import (
	"fmt"
	"testing"

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
	num1 := 4.6
	num2 := 9.8
	num3 := float64(7)
	var cases = []struct {
		name          string
		value         float64
		prefix        string
		expectedQuery string
		expectedValue string
	}{
		{
			name:          fmt.Sprintf("less than %v", num1),
			value:         num1,
			prefix:        "<",
			expectedQuery: "blah < $$",
			expectedValue: "4.60",
		},
		{
			name:          fmt.Sprintf("less than or equals to %v", num1),
			value:         num1,
			prefix:        "<=",
			expectedQuery: "blah <= $$",
			expectedValue: "4.60",
		},
		{
			name:          fmt.Sprintf("equals to %v", num1),
			value:         num1,
			prefix:        "=",
			expectedQuery: "blah = $$",
			expectedValue: "4.60",
		},
		{
			name:          fmt.Sprintf("equals to %v", num1),
			value:         num1,
			expectedQuery: "blah = $$",
			expectedValue: "4.60",
		},
		{
			name:          fmt.Sprintf("greater than or equals to %v", num1),
			value:         num1,
			prefix:        ">=",
			expectedQuery: "blah >= $$",
			expectedValue: "4.60",
		},
		{
			name:          fmt.Sprintf("greater than to %v", num2),
			value:         num2,
			prefix:        ">",
			expectedQuery: "blah > $$",
			expectedValue: "9.80",
		},
		{
			name:          fmt.Sprintf("integer equal %v", num3),
			value:         num3,
			expectedQuery: "blah = $$",
			expectedValue: "7",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := createNumericQuery("blah", c.prefix, c.value)
			assert.Equal(t, c.expectedQuery, actual.Query)
			assert.Equal(t, []interface{}{c.expectedValue}, actual.Values)
		})
	}
}
