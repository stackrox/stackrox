package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		name, src, expected string
		maxlen              int
		options             []TruncateOptions
	}{
		{
			name: "empty",
		},
		{
			name:     "single char",
			src:      "a",
			maxlen:   1,
			expected: "a",
		},
		{
			name:     "regular truncate",
			src:      "toolong",
			maxlen:   3,
			expected: "too",
		},
		{
			name:    "empty",
			options: []TruncateOptions{WordOriented{}},
		},
		{
			name:     "single char",
			src:      "a",
			maxlen:   1,
			options:  []TruncateOptions{WordOriented{}},
			expected: "a",
		},
		{
			name:     "single char space - no truncate",
			src:      " ",
			maxlen:   1,
			options:  []TruncateOptions{WordOriented{}},
			expected: " ",
		},
		{
			name:    "multiple char space - with truncate",
			src:     "  ",
			maxlen:  1,
			options: []TruncateOptions{WordOriented{}},
		},
		{
			name:     "separate words",
			src:      "hello there",
			maxlen:   2,
			options:  []TruncateOptions{WordOriented{}},
			expected: "he",
		},
		{
			name:     "truncate mid word",
			src:      "hello there",
			maxlen:   8,
			options:  []TruncateOptions{WordOriented{}},
			expected: "hello...",
		},
		{
			name:     "truncate at start of word",
			src:      "hello there",
			maxlen:   7,
			options:  []TruncateOptions{WordOriented{}},
			expected: "hell...",
		},
		{
			name:     "truncate very long word without maxcutoff",
			src:      "this isaverylongword",
			maxlen:   9,
			options:  []TruncateOptions{WordOriented{}},
			expected: "this...",
		},
		{
			name:     "truncate very long word with maxcutoff",
			src:      "this isaverylongword",
			maxlen:   15,
			options:  []TruncateOptions{WordOriented{MaxCutOff: 5}},
			expected: "this isavery...",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			truncated := Truncate(c.src, c.maxlen, c.options...)
			assert.Equal(t, c.expected, truncated)
			assert.LessOrEqualf(t, len(truncated), c.maxlen, "final truncate result %s did not respect max length", truncated)
		})
	}

}
