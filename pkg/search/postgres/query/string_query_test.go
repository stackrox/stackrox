package pgsearch

import (
	"testing"

	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringQuery(t *testing.T) {
	const colName = "blah"
	cases := []struct {
		value               string
		expectedWhereClause string
		expectedValues      []interface{}
		expectErr           bool
	}{
		{value: "test", expectedWhereClause: "blah = $$", expectedValues: []interface{}{"test"}},
		{value: "", expectedWhereClause: "blah = $$", expectedValues: []interface{}{""}},
	}
	for _, testCase := range cases {
		t.Run(testCase.value, func(t *testing.T) {
			actual, err := newStringQuery(&queryAndFieldContext{
				qualifiedColumnName: colName,
				value:               testCase.value,
				queryModifiers:      []pkgSearch.QueryModifier{pkgSearch.Equality},
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
