package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_indentAndWrap(t *testing.T) {
	expected := " \n" +
		"  This is some long text, that\n" +
		"   should be indented and\n" +
		"    \twrapped.\n" +
		"    There are multiple\n" +
		"    lines."

	sb := &strings.Builder{}
	_, err := makeWriter(sb, 30, 1, 2, 3, 4).WriteString(
		`
This is some long text, that should be indented and	wrapped.
There are multiple
lines.`)
	assert.NoError(t, err)
	assert.Equal(t, expected, sb.String())

	sb = &strings.Builder{}
	xw := makeWriter(sb, 30, 1, 2, 3, 4)
	_, _ = xw.WriteString("\nThis is")
	_, _ = xw.WriteString(" some long text")
	_, _ = xw.WriteString(", that should be indented ")
	_, _ = xw.WriteString("and\twrapped.\n")
	_, _ = xw.WriteString("There are multiple\nlines.")
	assert.Equal(t, expected, sb.String())

	cases := []struct {
		text     string
		padding  []int
		expected string
	}{
		{"single line", []int{0}, "single line"},
		{"two lines\nno padding", []int{0}, "two lines\nno padding"},
		{"two lines\nwith padding", []int{4}, "    two lines\n    with padding"},
		{"two lines\nwith different padding", []int{2, 4}, "  two lines\n    with different \n    padding"},
		{"three lines\nwith different\npadding", []int{2, 4, 1}, "  three lines\n    with different\n padding"},
		{"three lines\nwith some\npadding", []int{2, 4}, "  three lines\n    with some\n    padding"},
	}
	for _, c := range cases {
		sb := &strings.Builder{}
		_, err := makeWriter(sb, 20, c.padding...).WriteString(c.text)
		assert.NoError(t, err)
		assert.Equal(t, c.expected, sb.String())
	}
}

func Test_setIndent(t *testing.T) {
	t.Run("should respect updated indentation", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeWriter(sb, 20)
		_, _ = w.WriteString("text 0")
		w.setIndent(2, 4)
		_, _ = w.WriteString("text 2\n")
		_, _ = w.WriteString("text 4\n")
		_, _ = w.WriteString("text 4")
		w.setIndent()
		_, _ = w.WriteString("text 0\n")
		_, _ = w.WriteString("text 0\n")

		assert.Equal(t, "text 0  text 2\n    text 4\n    text 4text 0\ntext 0\n", sb.String())
	})

	t.Run("should not reset previously written line length", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeWriter(sb, 10)

		w.setIndent(2)               // 2
		_, _ = w.WriteString("... ") // +4=6
		w.setIndent(2)               // +2=8
		_, _ = w.WriteString(" .")   // +2=10
		assert.Equal(t, "  ...    .", sb.String())
	})

	t.Run("should wrap correctly", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeWriter(sb, 10)

		w.setIndent(2)
		_, _ = w.WriteString(".... ")
		w.setIndent(2)
		_, _ = w.WriteString(" ..")
		assert.Equal(t, "  ....    \n  ..", sb.String())
	})

}
