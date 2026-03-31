package phonehome

import (
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasHeader(t *testing.T) {

	t.Run("empty request", func(t *testing.T) {
		r := &RequestParams{}
		assert.True(t, r.HasHeader(nil))
		assert.False(t, r.HasHeader(map[glob.Pattern]glob.Pattern{"header": "value"}))
		assert.True(t, r.HasHeader(map[glob.Pattern]glob.Pattern{"header": NoHeaderOrAnyValue}))
	})

	rp := RequestParams{
		Headers: Headers(http.Header{
			"Empty": {},
			"One":   {"one"},
			"Two":   {"one", "two"},
		}),
	}

	tests := map[string]struct {
		headers  map[glob.Pattern]glob.Pattern
		expected bool
	}{
		"empty": {
			expected: true,
		},
		"empty not matching": {
			headers: map[glob.Pattern]glob.Pattern{
				"Empty": "with value",
			},
			expected: false,
		},
		"empty matching": {
			headers: map[glob.Pattern]glob.Pattern{
				"Empty": NoHeaderOrAnyValue,
			},
			expected: true,
		},
		"unknown empty": {
			headers: map[glob.Pattern]glob.Pattern{
				"Third": NoHeaderOrAnyValue,
			},
			expected: true,
		},
		"one": {
			headers: map[glob.Pattern]glob.Pattern{
				"One": "on?",
			},
			expected: true,
		},
		"one-two": {
			headers: map[glob.Pattern]glob.Pattern{
				"Two": "two",
			},
			expected: true,
		},
		"no match": {
			headers: map[glob.Pattern]glob.Pattern{
				"Three": "x*",
			},
			expected: false,
		},
		"one of multiple match": {
			headers: map[glob.Pattern]glob.Pattern{
				"One": "on?",
				"Two": "x",
			},
			expected: false,
		},
		"all of multiple match": {
			headers: map[glob.Pattern]glob.Pattern{
				"One": "on?",
				"Two": "two",
			},
			expected: true,
		},
		"one of multiple doesn't exist": {
			headers: map[glob.Pattern]glob.Pattern{
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
