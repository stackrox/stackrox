package stringutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinNonEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		parts    []string
		expected string
	}{
		{
			parts:    []string{},
			expected: "",
		},
		{
			parts:    []string{"", ""},
			expected: "",
		},
		{
			parts:    []string{"foo"},
			expected: "foo",
		},
		{
			parts:    []string{"", "foo", ""},
			expected: "foo",
		},
		{
			parts:    []string{"foo", "bar"},
			expected: "foo&bar",
		},
		{
			parts:    []string{"", "foo", "", "bar", ""},
			expected: "foo&bar",
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%+v", c.parts), func(t *testing.T) {
			actual := JoinNonEmpty("&", c.parts...)
			assert.Equal(t, c.expected, actual)
		})
	}
}
