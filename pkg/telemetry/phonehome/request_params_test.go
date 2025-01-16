package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasHeader(t *testing.T) {
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

	assert.True(t, (&RequestParams{}).HasHeader(nil))
	assert.False(t, (&RequestParams{}).HasHeader(map[string]Pattern{"header": "value"}))
	assert.True(t, (&RequestParams{}).HasHeader(map[string]Pattern{"header": NoHeaderOrAnyValuePattern}))

	tests := map[string]struct {
		patterns map[string]Pattern
		expected bool
	}{
		"empty": {
			expected: true,
		},
		"empty not matching": {
			patterns: map[string]Pattern{
				"empty": "with value",
			},
			expected: false,
		},
		"empty matching": {
			patterns: map[string]Pattern{
				"empty": NoHeaderOrAnyValuePattern,
			},
			expected: true,
		},
		"unknown empty": {
			patterns: map[string]Pattern{
				"third": NoHeaderOrAnyValuePattern,
			},
			expected: true,
		},
		"one": {
			patterns: map[string]Pattern{
				"one": "on?",
			},
			expected: true,
		},
		"one-two": {
			patterns: map[string]Pattern{
				"two": "two",
			},
			expected: true,
		},
		"no match": {
			patterns: map[string]Pattern{
				"three": "x*",
			},
			expected: false,
		},
		"one of multiple match": {
			patterns: map[string]Pattern{
				"one": "on?",
				"two": "x",
			},
			expected: false,
		},
		"all of multiple match": {
			patterns: map[string]Pattern{
				"one": "on?",
				"two": "two",
			},
			expected: true,
		},
		"one of multiple doesn't exist": {
			patterns: map[string]Pattern{
				"one":   "on?",
				"two":   "two",
				"three": "th*",
			},
			expected: false,
		},
	}
	for name, test := range tests {
		for _, pattern := range test.patterns {
			globCache[pattern], _ = pattern.compile()
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, rp.HasHeader(test.patterns))
		})
	}
}
