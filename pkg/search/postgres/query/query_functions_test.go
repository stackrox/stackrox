package pgsearch

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleExistenceQueries(t *testing.T) {
	const colName = "test_table.col"

	cases := map[string]struct {
		dataType      postgres.DataType
		value         string
		expectedQuery string
	}{
		"MatchAll on string should exclude empty strings": {
			dataType:      postgres.String,
			value:         pkgSearch.WildcardString,
			expectedQuery: "(test_table.col is not null AND test_table.col != '')",
		},
		"MatchNone on string should include empty strings": {
			dataType:      postgres.String,
			value:         pkgSearch.NullString,
			expectedQuery: "(test_table.col is null OR test_table.col = '')",
		},
		"MatchAll on numeric uses standard null check": {
			dataType:      postgres.Numeric,
			value:         pkgSearch.WildcardString,
			expectedQuery: "test_table.col is not null",
		},
		"MatchNone on numeric uses standard null check": {
			dataType:      postgres.Numeric,
			value:         pkgSearch.NullString,
			expectedQuery: "test_table.col is null",
		},
		"MatchAll on datetime uses standard null check": {
			dataType:      postgres.DateTime,
			value:         pkgSearch.WildcardString,
			expectedQuery: "test_table.col is not null",
		},
		"MatchNone on datetime uses standard null check": {
			dataType:      postgres.DateTime,
			value:         pkgSearch.NullString,
			expectedQuery: "test_table.col is null",
		},
		"MatchAll on integer uses standard null check": {
			dataType:      postgres.Integer,
			value:         pkgSearch.WildcardString,
			expectedQuery: "test_table.col is not null",
		},
		"MatchNone on integer uses standard null check": {
			dataType:      postgres.Integer,
			value:         pkgSearch.NullString,
			expectedQuery: "test_table.col is null",
		},
		"MatchAll on enum uses standard null check": {
			dataType:      postgres.Enum,
			value:         pkgSearch.WildcardString,
			expectedQuery: "test_table.col is not null",
		},
		"MatchNone on enum uses standard null check": {
			dataType:      postgres.Enum,
			value:         pkgSearch.NullString,
			expectedQuery: "test_table.col is null",
		},
		"MatchAll on UUID uses standard null check": {
			dataType:      postgres.UUID,
			value:         pkgSearch.WildcardString,
			expectedQuery: "test_table.col is not null",
		},
		"MatchNone on UUID uses standard null check": {
			dataType:      postgres.UUID,
			value:         pkgSearch.NullString,
			expectedQuery: "test_table.col is null",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			field := &pkgSearch.Field{
				FieldPath: "test.field",
			}
			qe, err := matchFieldQuery(colName, tc.dataType, field, nil, tc.value, false, time.Now())
			require.NoError(t, err)
			assert.Equal(t, tc.expectedQuery, qe.Where.Query)
		})
	}
}
