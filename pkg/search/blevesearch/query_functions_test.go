package blevesearch

import (
	"reflect"
	"testing"

	"github.com/blevesearch/bleve/search/query"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStringQuery(t *testing.T) {
	cases := []struct {
		query          string
		expectedString string
		hasError       bool
	}{
		{
			hasError: true,
		},
		{
			query:          "hello",
			expectedString: "hello",
		},
		{
			query:          "!hello",
			expectedString: "hello",
		},
		{
			query:          "!!hello",
			expectedString: "hello",
		},
		{
			query:          "!",
			expectedString: "!",
		},
		{
			query:          "r/lol",
			expectedString: "lol",
		},
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			q, err := matchFieldQuery(v1.SearchCategory_DEPLOYMENTS, "field", v1.SearchDataType_SEARCH_STRING, c.query)
			require.Equal(t, c.hasError, err != nil)
			if q != nil {
				switch typedQ := q.(type) {
				case *MatchPhrasePrefixQuery:
					assert.Equal(t, c.expectedString, typedQ.MatchPhrasePrefix)
				case *query.BooleanQuery:
					assert.Equal(t, c.expectedString, typedQ.MustNot.(*query.DisjunctionQuery).Disjuncts[0].(*MatchPhrasePrefixQuery).MatchPhrasePrefix)
				case *NegationQuery:
					assert.Equal(t, c.expectedString, typedQ.query.(*MatchPhrasePrefixQuery).MatchPhrasePrefix)
				case *query.RegexpQuery:
					assert.Equal(t, c.expectedString, typedQ.Regexp)
				default:
					t.Fatalf("Type '%s' not handled for query %q", reflect.TypeOf(q), c.query)
				}
			}
		})
	}
}
