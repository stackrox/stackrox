package regexutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchWholeString(t *testing.T) {
	cases := []struct {
		regex    string
		flags    Flags
		value    string
		expected bool
	}{
		{
			regex:    "",
			value:    "",
			expected: true,
		},
		{
			regex:    "",
			value:    "whatever",
			expected: true,
		},
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
			regex:    "bCD",
			value:    "bcd",
			expected: false,
		},
		{
			regex:    "bCD",
			value:    "bcd",
			flags:    Flags{CaseInsensitive: true},
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
		{
			regex:    "",
			value:    "abc",
			expected: true,
		},
		{
			regex:    "anacron|cron|crond|crontab",
			value:    "cron",
			expected: true,
		},
		{
			regex:    "anacron|cron|crond|crontab",
			value:    "crontab",
			expected: true,
		},
		{
			regex:    "anacron|cron|crond|crontab",
			value:    "cronta",
			expected: false,
		},
		{
			regex:    "abc$",
			value:    "abc",
			expected: true,
		},
		{
			regex:    "abc$",
			value:    "ab",
			expected: false,
		},
		{
			regex:    "^abc$",
			value:    "abc",
			expected: true,
		},
		{
			regex:    "^abc$",
			value:    "ab",
			expected: false,
		},
		{
			regex:    "^abc",
			value:    "abc",
			expected: true,
		},
		{
			regex:    "^abc",
			value:    "ab",
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s - %s - %v", c.regex, c.value, c.flags), func(t *testing.T) {
			m, err := CompileWholeStringMatcher(c.regex, c.flags)
			require.NoError(t, err)
			assert.Equal(t, c.expected, m.MatchWholeString(c.value))
		})
	}
}
