package regexutils

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchWholeString(t *testing.T) {
	cases := []struct {
		regex    string
		value    string
		expected bool
	}{
		{
			regex:    "abc",
			value:    "",
			expected: false,
		},
		{
			regex:    "bcd",
			value:    "bcd",
			expected: true,
		},
		{
			regex:    "bcd",
			value:    "abcd",
			expected: false,
		},
		{
			regex:    "abc",
			value:    "0abc",
			expected: false,
		},
		{
			regex:    "^$",
			value:    "",
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s - %s", c.regex, c.value), func(t *testing.T) {
			r := regexp.MustCompile(c.regex)
			assert.Equal(t, c.expected, MatchWholeString(r, c.value))
		})
	}
}
