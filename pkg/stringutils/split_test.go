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
		{"a", "", []string{"", "a"}},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			first, second := Split2(c.s, c.sep)
			assert.Equal(t, c.expected, []string{first, second})
		})
	}
}

func TestSplit2Last(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		sep      string
		expected []string
	}{
		{"Helo", "l", []string{"He", "o"}},
		{"Hello", "l", []string{"Hel", "o"}},
		{"Hello", "ll", []string{"He", "o"}},
		{"", "a", []string{"", ""}},
		{"a", "", []string{"a", ""}},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			first, second := Split2Last(c.s, c.sep)
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

func TestGetUpTo(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		sep      string
		expected string
	}{
		{"Hello", "", ""},
		{"Hello", "l", "He"},
		{"Hello", "/", "Hello"},
		{"", "", ""},
		{"hi/hello", "/", "hi"},
		{"/hello", "/", ""},
		{"hi/", "/", "hi"},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			assert.Equal(t, c.expected, GetUpTo(c.s, c.sep))
		})
	}
}

func TestGetAfter(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		sep      string
		expected string
	}{
		{"Hello", "", "Hello"},
		{"Hello", "l", "lo"},
		{"Hello", "/", "Hello"},
		{"", "", ""},
		{"hi/hello", "/", "hello"},
		{"/hello", "/", "hello"},
		{"hi/", "/", ""},
		{"h___e___l___l___o", "___", "e___l___l___o"},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			assert.Equal(t, c.expected, GetAfter(c.s, c.sep))
		})
	}
}

func TestGetAfterLast(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		sep      string
		expected string
	}{
		{"Hello", "", ""},
		{"Hello", "l", "o"},
		{"Hello", "/", "Hello"},
		{"", "", ""},
		{"hi/hello", "/", "hello"},
		{"/hello", "/", "hello"},
		{"hi/", "/", ""},
		{"h___e___l___l___o", "___", "o"},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			assert.Equal(t, c.expected, GetAfterLast(c.s, c.sep))
		})
	}
}

func TestGetBetween(t *testing.T) {
	for _, testCase := range []struct {
		s        string
		start    string
		end      string
		expected string
	}{
		{"", "", "", ""},
		{"hello", "", "", ""},
		{"(", "(", ")", ""},
		{"((", "(", "(", ""},
		{"(abc(", "(", "(", "abc"},
		{"()", "(", ")", ""},
		{")", "(", ")", ""},
		{"(abc)", "(", ")", "abc"},
		{"(abc)(abc)", "(", ")", "abc"},
		{"def(abc)def", "(", ")", "abc"},
		{"(((abc))", "(((", "))", "abc"},
	} {
		c := testCase
		t.Run(fmt.Sprintf("str=%s,start=%s,end=%s", c.s, c.start, c.end), func(t *testing.T) {
			assert.Equal(t, c.expected, GetBetween(c.s, c.start, c.end))
		})
	}
}
