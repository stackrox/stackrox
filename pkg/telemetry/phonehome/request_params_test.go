package phonehome

import (
	"testing"

	"github.com/stackrox/rox/pkg/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasHeader(t *testing.T) {

	t.Run("empty request", func(t *testing.T) {
		r := &RequestParams{}
		assert.True(t, r.HasHeader(nil))
		assert.False(t, r.HasHeader(map[string]glob.Pattern{"header": "value"}))
		assert.True(t, r.HasHeader(map[string]glob.Pattern{"header": NoHeaderOrAnyValue}))
	})

	rp := RequestParams{
		Headers: func(s string) []string {
			headers := map[string][]string{
				"empty": {},
				"one":   {"one"},
				"two":   {"one", "two"},
			}
			return headers[s]
		},
	}

	tests := map[string]struct {
		headers  map[string]glob.Pattern
		expected bool
	}{
		"empty": {
			expected: true,
		},
		"empty not matching": {
			headers: map[string]glob.Pattern{
				"empty": "with value",
			},
			expected: false,
		},
		"empty matching": {
			headers: map[string]glob.Pattern{
				"empty": NoHeaderOrAnyValue,
			},
			expected: true,
		},
		"unknown empty": {
			headers: map[string]glob.Pattern{
				"third": NoHeaderOrAnyValue,
			},
			expected: true,
		},
		"one": {
			headers: map[string]glob.Pattern{
				"one": "on?",
			},
			expected: true,
		},
		"one-two": {
			headers: map[string]glob.Pattern{
				"two": "two",
			},
			expected: true,
		},
		"no match": {
			headers: map[string]glob.Pattern{
				"three": "x*",
			},
			expected: false,
		},
		"one of multiple match": {
			headers: map[string]glob.Pattern{
				"one": "on?",
				"two": "x",
			},
			expected: false,
		},
		"all of multiple match": {
			headers: map[string]glob.Pattern{
				"one": "on?",
				"two": "two",
			},
			expected: true,
		},
		"one of multiple doesn't exist": {
			headers: map[string]glob.Pattern{
				"one":   "on?",
				"two":   "two",
				"three": "th*",
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
