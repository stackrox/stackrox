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
			maxlen:   7,
			options:  []TruncateOptions{WordOriented{}},
			expected: "hello...",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, Truncate(c.src, c.maxlen, c.options...))
		})
	}

}
