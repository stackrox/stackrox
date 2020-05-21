package stringutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit2(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		sep      string
		expected []string
	}{
		{"Helo", "l", []string{"He", "o"}},
		{"Hello", "l", []string{"He", "lo"}},
		{"Hello", "ll", []string{"He", "o"}},
		{"", "a", []string{"", ""}},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			first, second := Split2(c.s, c.sep)
			assert.Equal(t, c.expected, []string{first, second})
		})
	}
}

func TestSplitNPadded(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		sep      string
		n        int
		expected []string
	}{
		// Ensure it acts like Split2 when n == 2.
		{"Helo", "l", 2, []string{"He", "o"}},
		{"Hello", "l", 2, []string{"He", "lo"}},
		{"Hello", "ll", 2, []string{"He", "o"}},
		{"", "a", 2, []string{"", ""}},

		{"Helo", "l", 3, []string{"He", "o", ""}},
		{"Hello", "l", 5, []string{"He", "", "o", "", ""}},
		{"", "a", 1, []string{""}},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			assert.Equal(t, c.expected, SplitNPadded(c.s, c.sep, c.n))
		})
	}
}
