package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsWhitespace(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected bool
	}{
		{
			name:     "OneWord",
			s:        "OneWOORDD",
			expected: false,
		},
		{
			name:     "OneWordWithPunc",
			s:        "OneWOSA1!@$!@T!@%ffaksf124@~",
			expected: false,
		},
		{
			name:     "Empty",
			s:        "",
			expected: false,
		},
		{
			name:     "Sentence",
			s:        "This is a sentence",
			expected: true,
		},
		{
			name:     "One space only",
			s:        " ",
			expected: true,
		},
		{
			name:     "Tabs",
			s:        "This\tis\ttab\tseparated",
			expected: true,
		},
		{
			name:     "Newlines",
			s:        "This\nis\nnewline",
			expected: true,
		},
		{
			name:     "All of the whitespace",
			s:        "  \t\t\n\n",
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := ContainsWhitespace(c.s)
			assert.Equal(t, c.expected, out)
		})
	}
}
