package blevesearch

import (
	"fmt"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	ID     string       `json:"id"`
	Nested []TestStruct `json:"nested"`
	Key    string       `json:"key"`
}

func checkResults(t *testing.T, idToArrayPositions map[string][]search.ArrayPositions, field string, result *bleve.SearchResult) {
	// We check that every hit has a corresponding idToArrayPosition so checking length ensures that they are equal
	assert.Equal(t, len(idToArrayPositions), len(result.Hits))
	for _, h := range result.Hits {
		arrayPositions := idToArrayPositions[h.ID]
		assert.NotEmpty(t, arrayPositions)
		tlm := h.Locations[field]
		for _, locations := range tlm {
			// We check that every loc has a corresponding array position so a check for length will guarantee they are the same
			assert.Equal(t, len(arrayPositions), len(locations))
			for _, loc := range locations {
				var match bool
				for _, arrayPos := range arrayPositions {
					if arrayPos.Equals(loc.ArrayPositions) {
						match = true
						break
					}
				}
				assert.True(t, match)
			}
		}
	}
}

func TestNegation(t *testing.T) {
	im := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(im)
	require.NoError(t, err)

	structs := []*TestStruct{
		{
			ID: "1",
			Nested: []TestStruct{
				{
					Nested: []TestStruct{
						{
							Key: "AA",
						},
					},
				},
			},
		},
		{
			ID: "2",
			Nested: []TestStruct{
				{
					Nested: []TestStruct{
						{
							Key: "BB",
						},
					},
				},
				{
					Nested: []TestStruct{
						{
							Key: "AA",
						},
					},
				},
				{
					Nested: []TestStruct{
						{
							Key: "CC",
						},
						{
							Key: "BB",
						},
						{
							Key: "AA",
						},
					},
				},
				{
					Nested: []TestStruct{
						{
							Key: "CC",
						},
					},
				},
			},
		},
		{
			ID: "3",
			Nested: []TestStruct{
				{
					Nested: []TestStruct{
						{
							Key: "BB",
						},
					},
				},
				{
					Nested: []TestStruct{
						{
							Key: "BB",
						},
					},
				},
			},
		},
	}
	for _, s := range structs {
		require.NoError(t, idx.Index(s.ID, s))
	}

	cases := []struct {
		q               query.Query
		required        bool
		expectedResults map[string][]search.ArrayPositions
	}{
		{
			q: NewMatchPhrasePrefixQuery("nested.nested.key", "AA"),
			expectedResults: map[string][]search.ArrayPositions{
				"3": {
					{
						0, 0,
					},
					{
						1, 0,
					},
				},
			},
		},
		{
			q:        NewMatchPhrasePrefixQuery("nested.nested.key", "AA"),
			required: true,
			expectedResults: map[string][]search.ArrayPositions{
				"2": {
					{
						0, 0,
					},
					{
						2, 0,
					},
					{
						2, 1,
					},
					{
						3, 0,
					},
				},
				"3": {
					{
						0, 0,
					},
					{
						1, 0,
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s-%t", c.q, c.required), func(t *testing.T) {
			nq := NewNegationQuery(bleve.NewMatchAllQuery(), c.q, c.required)
			req := bleve.NewSearchRequest(nq)
			req.IncludeLocations = true
			result, err := idx.Search(req)
			require.NoError(t, err)

			checkResults(t, c.expectedResults, "nested.nested.key", result)
		})
	}
}
