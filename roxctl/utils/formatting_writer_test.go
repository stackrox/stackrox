package utils

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_indentAndWrap(t *testing.T) {
	expected := "\n" +
		"  This is some long text, that\n" +
		"   should be indented and\n" +
		"    \twrapped.\n" +
		"    There are multiple\n" +
		"    lines."

	sb := &strings.Builder{}
	_, err := makeFormattingWriter(sb, 30, defaultTabWidth, 1, 2, 3, 4).WriteString(
		`
This is some long text, that should be indented and	wrapped.
There are multiple
lines.`)
	assert.NoError(t, err)
	assert.Equal(t, expected, sb.String())

	sb = &strings.Builder{}
	xw := makeFormattingWriter(sb, 30, defaultTabWidth, 1, 2, 3, 4)
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
		_, err := makeFormattingWriter(sb, 20, defaultTabWidth, c.padding...).WriteString(c.text)
		assert.NoError(t, err)
		assert.Equal(t, c.expected, sb.String())
	}
}

func Test_setIndent(t *testing.T) {
	t.Run("should respect updated indentation", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeFormattingWriter(sb, 20, defaultTabWidth)
		_, _ = w.WriteString("text 0")
		w.SetIndent(2, 4)
		_, _ = w.WriteString("text 2\n")
		_, _ = w.WriteString("text 4\n")
		_, _ = w.WriteString("text 4")
		w.SetIndent()
		_, _ = w.WriteString("text 0\n")
		_, _ = w.WriteString("text 0\n")

		assert.Equal(t, "text 0  text 2\n    text 4\n    text 4text 0\ntext 0\n", sb.String())
	})

	t.Run("should not reset previously written line length", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeFormattingWriter(sb, 10, defaultTabWidth)

		w.SetIndent(2)               // 2
		_, _ = w.WriteString("... ") // +4=6
		w.SetIndent(2)               // +2=8
		_, _ = w.WriteString(" .")   // +2=10
		assert.Equal(t, "  ...    .", sb.String())
	})

	t.Run("should wrap correctly", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeFormattingWriter(sb, 10, defaultTabWidth)

		w.SetIndent(2)
		_, _ = w.WriteString(".... ")
		w.SetIndent(2)
		_, _ = w.WriteString(" ..")
		assert.Equal(t, "  ....    \n  ..", sb.String())
	})

	t.Run("negative indent should tab", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeFormattingWriter(sb, 20, defaultTabWidth)

		_, _ = w.WriteString("\n")
		w.SetIndent(15)
		_, _ = w.WriteString(">")
		w.SetIndent(15)
		_, _ = w.WriteString("|\n")
		w.SetIndent(2)
		_, _ = w.WriteString(">")
		w.SetIndent(-10)
		_, _ = w.WriteString("|\n|\n")
		w.SetIndent(2)
		_, _ = w.WriteString(">>>>>>>>>")
		w.SetIndent(-10)
		_, _ = w.WriteString("|\n")
		w.SetIndent(2)
		_, _ = w.WriteString(">>>>>>>>")
		w.SetIndent(-10)
		_, _ = w.WriteString("|\n")
		w.SetIndent(2)
		_, _ = w.WriteString(">>>>>>>")
		w.SetIndent(-10)
		_, _ = w.WriteString("|\n|")
		w.SetIndent(-15)
		_, _ = w.WriteString("|\n|")
		assert.Equal(t, `
               >
               |
  >       |
          |
  >>>>>>>>>
          |
  >>>>>>>>
          |
  >>>>>>> |
          |    |
               |`, sb.String())
	})

	t.Run("No padding before linebreak", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeFormattingWriter(sb, 14, defaultTabWidth)

		w.SetIndent(4)
		_, _ = w.WriteString("\n")
		_, _ = w.WriteString("Blue trees arent real")
		assert.Equal(t, `
    Blue trees
    arent real`, sb.String())
	})

	t.Run("single LN when string is longer than width", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeFormattingWriter(sb, 5, defaultTabWidth)

		w.SetIndent(0, 3)
		_, _ = w.WriteString("abc ")
		_, _ = w.WriteString("def")
		w.SetIndent(4)
		_, _ = w.WriteString("ghi")
		assert.Equal(t, "abc \n   def\n    ghi", sb.String())
	})
}

type sbWithErrors struct {
	strings.Builder
	fail func(n int, s string) bool
}

func (sbe *sbWithErrors) WriteString(s string) (int, error) {
	if sbe.fail(sbe.Builder.Len(), s) {
		return 0, errors.New("test error")
	}
	return sbe.Builder.WriteString(s)
}

func Test_writeErrors(t *testing.T) {
	t.Run("simple string", func(t *testing.T) {
		sb := &sbWithErrors{fail: func(n int, s string) bool {
			return n+len(s) > 5
		}}
		w := makeFormattingWriter(sb, 10, defaultTabWidth)

		n, err := w.WriteString("abcde")
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		n, err = w.WriteString("fgh")
		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})
	t.Run("write error on wrapping", func(t *testing.T) {
		sb := &sbWithErrors{fail: func(_ int, s string) bool {
			return s == "\n"
		}}
		w := makeFormattingWriter(sb, 10, defaultTabWidth)
		n, err := w.WriteString("abcde fghij")
		assert.Error(t, err)
		assert.Equal(t, 6, n)
	})
	t.Run("write error on padding", func(t *testing.T) {
		sb := &sbWithErrors{fail: func(_ int, s string) bool {
			return s == "  "
		}}
		w := makeFormattingWriter(sb, 10, defaultTabWidth, 0, 2)
		n, err := w.WriteString("abcde fghij")
		assert.Error(t, err)
		assert.Equal(t, 6, n)
	})
	t.Run("write error on initial padding", func(t *testing.T) {
		sb := &sbWithErrors{fail: func(_ int, s string) bool {
			return s == "  "
		}}
		w := makeFormattingWriter(sb, 10, defaultTabWidth, 2)
		n, err := w.WriteString("a")
		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})
}
