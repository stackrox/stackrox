package query

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/mapeval"
	"gotest.tools/assert"
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
				ShouldNotContain("a", ""),
			},
			expected: mapeval.ShouldNotMatchMarker + "a=",
		},
		{
			desc: "Simple should not contain query",
			queries: []string{
				ShouldNotContain("a", "b"),
			},
			expected: mapeval.ShouldNotMatchMarker + "a=b",
		},
		{
			desc: "Simple should contain query",
			queries: []string{
				ShouldContain("a", ""),
			},
			expected: "a=",
		},
		{
			desc: "Simple should not contain query",
			queries: []string{
				ShouldNotContain("", "a"),
			},
			expected: mapeval.ShouldNotMatchMarker + "=a",
		},
		{
			desc: "Simple should contain query",
			queries: []string{
				ShouldContain("", "a"),
			},
			expected: "=a",
		},
		{
			desc: "Simple should contain query",
			queries: []string{
				ShouldContain("a", "b"),
			},
			expected: "a=b",
		},
		{
			desc: "Simple disjunction query",
			queries: []string{
				ShouldContain("a", "b"),
				ShouldContain("a", "c"),
				ShouldNotContain("", "a"),
			},
			expected: "a=b" + mapeval.DisjunctionMarker + "a=c" + mapeval.DisjunctionMarker + mapeval.ShouldNotMatchMarker + "=a",
		},
		{
			desc: "Simple conjunction query",
			queries: []string{
				ShouldMatchIfAllOf(ShouldContain("a", "b"), ShouldNotContain("b", "2")),
			},
			expected: "a=b" + mapeval.ConjunctionMarker + mapeval.ShouldNotMatchMarker + "b=2",
		},
		{
			desc: "Simple conjunction and disjunction query",
			queries: []string{
				ShouldMatchIfAllOf(ShouldContain("a", "b"), ShouldNotContain("b", "2")),
				ShouldContain("a", "b"),
				ShouldNotContain("", "a"),
			},
			expected: "a=b" + mapeval.ConjunctionMarker + mapeval.ShouldNotMatchMarker + "b=2" + mapeval.DisjunctionMarker + "a=b" + mapeval.DisjunctionMarker + mapeval.ShouldNotMatchMarker + "=a",
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			resQ := CompileMapQuery(c.queries...)
			assert.Equal(t, resQ, c.expected)
		})

	}
}
