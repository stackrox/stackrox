package pgsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnumQuery(t *testing.T) {
	const colName = "blah"
	cases := []struct {
		values         []int32
		expectedQuery  string
		expectedValues []any
		expectErr      bool
	}{
		{values: nil, expectedQuery: "", expectedValues: nil, expectErr: true},
		{values: []int32{1}, expectedQuery: "blah = $$", expectedValues: []any{"1"}},
		{values: []int32{1, 2}, expectedQuery: "blah IN ($$, $$)", expectedValues: []any{"1", "2"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.expectedQuery, func(t *testing.T) {
			actual, err := enumEquality(colName, testCase.values)
			if testCase.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedQuery, actual.Query)
			assert.Equal(t, testCase.expectedValues, actual.Values)
		})
	}
}
