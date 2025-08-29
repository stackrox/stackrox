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
			assert.Equal(t, c.expected, m.MatchString(c.value))
		})
	}
}

func TestMatchContainsString(t *testing.T) {
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
			expected: true,
		},
		{
			regex:    "abc",
			value:    "0abc",
			expected: true,
		},
		{
			regex:    "abc",
			value:    "0abc123",
			expected: true,
		},
		{
			regex:    "abc",
			value:    "xyz",
			expected: false,
		},
		{
			regex:    "test",
			value:    "this is a test string",
			expected: true,
		},
		{
			regex:    "test",
			value:    "testing 123",
			expected: true,
		},
		{
			regex:    "^abc",
			value:    "abc123",
			expected: true,
		},
		{
			regex:    "^abc",
			value:    "123abc",
			expected: false,
		},
		{
			regex:    "abc$",
			value:    "123abc",
			expected: true,
		},
		{
			regex:    "abc$",
			value:    "abc123",
			expected: false,
		},
		{
			regex:    "cron",
			value:    "anacron service",
			expected: true,
		},
		{
			regex:    "cron",
			value:    "crontab entry",
			expected: true,
		},
		{
			regex:    "cron",
			value:    "docker daemon",
			expected: false,
		},
		{
			regex:    "\\d+",
			value:    "version 123",
			expected: true,
		},
		{
			regex:    "\\d+",
			value:    "no numbers here",
			expected: false,
		},
		{
			regex:    "[a-z]+",
			value:    "ABC123def",
			expected: true,
		},
		{
			regex:    "[A-Z]+",
			value:    "abc123DEF",
			expected: true,
		},
		{
			regex:    "[A-Z]+",
			value:    "abc123def",
			expected: false,
		},
		{
			regex:    "[A-Z]+",
			value:    "abc123DEF",
			flags:    Flags{CaseInsensitive: true},
			expected: true,
		},
		{
			regex:    "cat|dog",
			value:    "I have a cat",
			expected: true,
		},
		{
			regex:    "cat|dog",
			value:    "I have a dog",
			expected: true,
		},
		{
			regex:    "cat|dog",
			value:    "I have a bird",
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s - %s - %v", c.regex, c.value, c.flags), func(t *testing.T) {
			m, err := CompileContainsStringMatcher(c.regex, c.flags)
			require.NoError(t, err)
			assert.Equal(t, c.expected, m.MatchString(c.value))
		})
	}
}
