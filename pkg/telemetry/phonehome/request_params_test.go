package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchHeaders(t *testing.T) {

	t.Run("empty request", func(t *testing.T) {
		r := &RequestParams{}
		assert.Equal(t, Headers{}, r.MatchHeaders(nil))
		assert.Nil(t, r.MatchHeaders(GlobMap{"header": "value"}))
		assert.Equal(t, Headers{}, r.MatchHeaders(GlobMap{"header": NoHeaderOrAnyValue}))
	})

	rp := RequestParams{
		Headers: Headers{
			"Empty": {},
			"One":   {"one"},
			"Two":   {"one", "two"},
		},
	}

	tests := map[string]struct {
		headers  GlobMap
		expected Headers
	}{
		"empty": {
			expected: Headers{},
		},
		"empty not matching": {
			headers: GlobMap{
				"Empty": "with value",
			},
			expected: nil,
		},
		"empty matching": {
			headers: GlobMap{
				"Empty": NoHeaderOrAnyValue,
			},
			expected: Headers{"Empty": {}},
		},
		"unknown empty": {
			headers: GlobMap{
				"Third": NoHeaderOrAnyValue,
			},
			expected: Headers{},
		},
		"one": {
			headers: GlobMap{
				"One": "on?",
			},
			expected: Headers{"One": {"one"}},
		},
		"one-two": {
			headers: GlobMap{
				"Two": "two",
			},
			expected: Headers{"Two": {"two"}},
		},
		"no match": {
			headers: GlobMap{
				"Three": "x*",
			},
			expected: nil,
		},
		"one of multiple match": {
			headers: GlobMap{
				"One": "on?",
				"Two": "x",
			},
			expected: nil,
		},
		"all of multiple match": {
			headers: GlobMap{
				"One": "on?",
				"Two": "two",
			},
			expected: Headers{"One": {"one"}, "Two": {"two"}},
		},
		"one of multiple doesn't exist": {
			headers: GlobMap{
				"One":   "on?",
				"Two":   "two",
				"Three": "th*",
			},
			expected: nil,
		},
	}
	for name, test := range tests {
		require.NoError(t, (&APICallCampaignCriterion{Headers: test.headers}).Compile())

		t.Run(name, func(t *testing.T) {
			h := rp.MatchHeaders(test.headers)
			assert.Equal(t, test.expected, h)
		})
	}
}
