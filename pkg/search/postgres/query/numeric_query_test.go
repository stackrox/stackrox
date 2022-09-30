package pgsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNumericPrefix(t *testing.T) {
	cases := []struct {
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

func TestNumericQuery(t *testing.T) {
	const colName = "blah"
	cases := []struct {
		value               string
		expectedWhereClause string
		expectedValues      []interface{}
		expectErr           bool
	}{
		{value: "<4.60", expectedWhereClause: "blah < $$", expectedValues: []interface{}{"4.6"}},
		{value: "<=4.60", expectedWhereClause: "blah <= $$", expectedValues: []interface{}{"4.6"}},
		{value: "=4.60", expectedWhereClause: "blah = $$", expectedValues: []interface{}{"4.6"}},
		{value: "==4.60", expectedWhereClause: "blah = $$", expectedValues: []interface{}{"4.6"}},
		{value: ">=4.60", expectedWhereClause: "blah >= $$", expectedValues: []interface{}{"4.6"}},
		{value: ">9.80", expectedWhereClause: "blah > $$", expectedValues: []interface{}{"9.8"}},
		{value: "7", expectedWhereClause: "blah = $$", expectedValues: []interface{}{"7"}},
		{value: ">1", expectedWhereClause: "blah > $$", expectedValues: []interface{}{"1"}},
		{value: ">4294967295", expectedWhereClause: "blah > $$", expectedValues: []interface{}{"4294967295"}},
		{value: "-1", expectedWhereClause: "blah = $$", expectedValues: []interface{}{"-1"}},
		{value: "1-2", expectedWhereClause: "(blah > $$) AND (blah < $$)", expectedValues: []interface{}{"1", "2"}},
		{value: "3294967295-4294967295", expectedWhereClause: "(blah > $$) AND (blah < $$)", expectedValues: []interface{}{"3294967295", "4294967295"}},
		{value: "-1--2", expectErr: true},
		{value: "-2--1", expectedWhereClause: "(blah > $$) AND (blah < $$)", expectedValues: []interface{}{"-2", "-1"}},
		{value: "-2.9124--1.2", expectedWhereClause: "(blah > $$) AND (blah < $$)", expectedValues: []interface{}{"-2.91", "-1.2"}},
		{value: "-2-1", expectedWhereClause: "(blah > $$) AND (blah < $$)", expectedValues: []interface{}{"-2", "1"}},
		{value: "1.2-2.992", expectedWhereClause: "(blah > $$) AND (blah < $$)", expectedValues: []interface{}{"1.2", "2.99"}},
		{value: "1-1", expectErr: true},
		{value: "2-1", expectErr: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.value, func(t *testing.T) {
			actual, err := newNumericQuery(&queryAndFieldContext{
				qualifiedColumnName: colName,
				value:               testCase.value,
			})
			if testCase.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedWhereClause, actual.Where.Query)
			assert.Equal(t, testCase.expectedValues, actual.Where.Values)
		})
	}
}
