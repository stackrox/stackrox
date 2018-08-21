package blevesearch

import (
	"reflect"
	"testing"

	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetStringToSearch(t *testing.T) {
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
			query:          "!",
			expectedString: "!",
		},
		{
			query:          "/lol",
			expectedString: "lol",
		},
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			q, err := getQueryToSearch(v1.SearchCategory_DEPLOYMENTS, "field", c.query)
			assert.Equal(t, c.hasError, err != nil)
			if q != nil {
				switch q.(type) {
				case *MatchPhrasePrefixQuery:
					assert.Equal(t, c.expectedString, q.(*MatchPhrasePrefixQuery).MatchPhrasePrefix)
				case *query.BooleanQuery:
					assert.Equal(t, c.expectedString, q.(*query.BooleanQuery).MustNot.(*query.DisjunctionQuery).Disjuncts[0].(*MatchPhrasePrefixQuery).MatchPhrasePrefix)
				case *query.RegexpQuery:
					assert.Equal(t, c.expectedString, q.(*query.RegexpQuery).Regexp)
				default:
					t.Fatalf("Type '%s' not handled", reflect.TypeOf(q))
				}
			}
		})

	}
}
