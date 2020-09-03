package stringutils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLongestCommonPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b     string
		expected string
	}{
		{
			a:        "",
			b:        "",
			expected: "",
		},
		{
			a:        "",
			b:        "foo",
			expected: "",
		},
		{
			a:        "foo",
			b:        "bar",
			expected: "",
		},
		{
			a:        "foo",
			b:        "foobar",
			expected: "foo",
		},
		{
			a:        "fooqux",
			b:        "foobar",
			expected: "foo",
		},
		{
			a:        "foobar",
			b:        "foobar",
			expected: "foobar",
		},
		{
			a:        "fööbar",
			b:        "fööqux",
			expected: "föö",
		},
		{
			a:        "föö",
			b:        "füü",
			expected: "f\xc3",
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s %s", c.a, c.b), func(t *testing.T) {
			lcp := LongestCommonPrefix(c.a, c.b)
			assert.Equal(t, c.expected, lcp)
			assert.Equal(t, LongestCommonPrefix(c.b, c.a), lcp)

			assert.Truef(t, strings.HasPrefix(c.a, lcp), "%s is not A prefix of %s", lcp, c.a)
			assert.Truef(t, strings.HasPrefix(c.b, lcp), "%s is not A prefix of %s", lcp, c.b)

			assert.True(t, len(lcp) == len(c.a) || len(lcp) == len(c.b) || c.a[len(lcp)] != c.b[len(lcp)])
		})
	}
}

func TestLongestCommonPrefixUTF8(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b     string
		expected string
	}{
		{
			a:        "",
			b:        "",
			expected: "",
		},
		{
			a:        "",
			b:        "foo",
			expected: "",
		},
		{
			a:        "foo",
			b:        "bar",
			expected: "",
		},
		{
			a:        "foo",
			b:        "foobar",
			expected: "foo",
		},
		{
			a:        "fooqux",
			b:        "foobar",
			expected: "foo",
		},
		{
			a:        "foobar",
			b:        "foobar",
			expected: "foobar",
		},
		{
			a:        "fööbar",
			b:        "fööqux",
			expected: "föö",
		},
		{
			a:        "föö",
			b:        "füü",
			expected: "f", // this is different from LongestCommonPrefix
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s %s", c.a, c.b), func(t *testing.T) {
			lcp := LongestCommonPrefixUTF8(c.a, c.b)
			assert.Equal(t, c.expected, lcp)
			assert.Equal(t, LongestCommonPrefixUTF8(c.b, c.a), lcp)

			assert.Truef(t, strings.HasPrefix(c.a, lcp), "%s is not A prefix of %s", lcp, c.a)
			assert.Truef(t, strings.HasPrefix(c.b, lcp), "%s is not A prefix of %s", lcp, c.b)

			assert.True(t, len(lcp) == len(c.a) || len(lcp) == len(c.b) || []rune(c.a)[len([]rune(lcp))] != []rune(c.b)[len([]rune(lcp))])
		})
	}
}
