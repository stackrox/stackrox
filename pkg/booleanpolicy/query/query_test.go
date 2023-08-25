package query

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/mapeval"
	"github.com/stretchr/testify/assert"
)

type testcase struct {
	desc     string
	queries  []string
	expected string
}

func TestMapQueries(t *testing.T) {
	testCases := []testcase{
		{
			desc: "Simple should not contain query",
			queries: []string{
				MapShouldNotContain("a", ""),
			},
			expected: mapeval.ShouldNotMatchMarker + "a=",
		},
		{
			desc: "Simple should not contain query",
			queries: []string{
				MapShouldNotContain("a", "b"),
			},
			expected: mapeval.ShouldNotMatchMarker + "a=b",
		},
		{
			desc: "Simple should contain query",
			queries: []string{
				MapShouldContain("a", ""),
			},
			expected: "a=",
		},
		{
			desc: "Simple should not contain query",
			queries: []string{
				MapShouldNotContain("", "a"),
			},
			expected: mapeval.ShouldNotMatchMarker + "=a",
		},
		{
			desc: "Simple should contain query",
			queries: []string{
				MapShouldContain("", "a"),
			},
			expected: "=a",
		},
		{
			desc: "Simple should contain query",
			queries: []string{
				MapShouldContain("a", "b"),
			},
			expected: "a=b",
		},
		{
			desc: "Simple disjunction query",
			queries: []string{
				MapShouldContain("a", "b"),
				MapShouldContain("a", "c"),
				MapShouldNotContain("", "a"),
			},
			expected: "a=b" + mapeval.DisjunctionMarker + "a=c" + mapeval.DisjunctionMarker + mapeval.ShouldNotMatchMarker + "=a",
		},
		{
			desc: "Simple conjunction query",
			queries: []string{
				MapShouldMatchAllOf(MapShouldContain("a", "b"), MapShouldNotContain("b", "2")),
			},
			expected: "a=b" + mapeval.ConjunctionMarker + mapeval.ShouldNotMatchMarker + "b=2",
		},
		{
			desc: "Simple conjunction and disjunction query",
			queries: []string{
				MapShouldMatchAllOf(MapShouldContain("a", "b"), MapShouldNotContain("b", "2")),
				MapShouldContain("a", "b"),
				MapShouldNotContain("", "a"),
			},
			expected: "a=b" + mapeval.ConjunctionMarker + mapeval.ShouldNotMatchMarker + "b=2" + mapeval.DisjunctionMarker + "a=b" + mapeval.DisjunctionMarker + mapeval.ShouldNotMatchMarker + "=a",
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			resQ := MapShouldMatchAnyOf(c.queries...)
			assert.Equal(t, c.expected, resQ)
		})

	}
}
