package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasHeader(t *testing.T) {

	t.Run("empty request", func(t *testing.T) {
		r := &RequestParams{}
		assert.True(t, r.HasHeader(nil))
		assert.False(t, r.HasHeader(GlobMap{"header": "value"}))
		assert.True(t, r.HasHeader(GlobMap{"header": NoHeaderOrAnyValue}))
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
		expected bool
	}{
		"empty": {
			expected: true,
		},
		"empty not matching": {
			headers: GlobMap{
				"Empty": "with value",
			},
			expected: false,
		},
		"empty matching": {
			headers: GlobMap{
				"Empty": NoHeaderOrAnyValue,
			},
			expected: true,
		},
		"unknown empty": {
			headers: GlobMap{
				"Third": NoHeaderOrAnyValue,
			},
			expected: true,
		},
		"one": {
			headers: GlobMap{
				"One": "on?",
			},
			expected: true,
		},
		"one-two": {
			headers: GlobMap{
				"Two": "two",
			},
			expected: true,
		},
		"no match": {
			headers: GlobMap{
				"Three": "x*",
			},
			expected: false,
		},
		"one of multiple match": {
			headers: GlobMap{
				"One": "on?",
				"Two": "x",
			},
			expected: false,
		},
		"all of multiple match": {
			headers: GlobMap{
				"One": "on?",
				"Two": "two",
			},
			expected: true,
		},
		"one of multiple doesn't exist": {
			headers: GlobMap{
				"One":   "on?",
				"Two":   "two",
				"Three": "th*",
			},
			expected: false,
		},
	}
	for name, test := range tests {
		require.NoError(t, (&APICallCampaignCriterion{Headers: test.headers}).Compile())

		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, rp.HasHeader(test.headers))
		})
	}
}
