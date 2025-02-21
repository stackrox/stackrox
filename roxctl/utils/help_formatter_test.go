package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func Test_indentAndWrap(t *testing.T) {
	expected := " \n" +
		"  This is some long text, that\n" +
		"   should be indented and\t\n" +
		"    wrapped.\n" +
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

func TestFormatHelp(t *testing.T) {
	c := &cobra.Command{
		Use:     "test",
		Short:   "short description",
		Long:    "long description\nmultiline",
		Aliases: []string{"t1", "t2"},
		Example: "test this:\ntest command",
	}
	c.PersistentFlags().Bool("bflag0", false, "persistent bool flag with default false value")
	c.Flags().Bool("bflag1", true, "bool with default true value")
	c.Flags().String("sflag", "defstr", "string flag with default value")
	c.Flags().Int("iflag", 10, "integer flag")
	c.Flags().Duration("dflag", time.Minute, "duration flag")
	c.Flags().StringP("spflag", "s", "", "string flag with shorthand")
	c.Flags().StringArray("sarray", []string{"a", "b"}, "string array flag with default array value")

	subcommand := &cobra.Command{
		Use:   "sub-test",
		Short: "short sub test description",
		Long:  "long sub test description\nmultiline",
	}
	subcommand.Flags().Int64("i64", int64(42), "int64 flag with default value")

	c.AddCommand(subcommand)

	t.Run("test main command", func(t *testing.T) {
		sb := &strings.Builder{}
		sb.WriteRune('\n')
		c.SetOut(sb)

		FormatHelp(c, nil)
		assert.Equal(t, `
short description

long description
multiline

Aliases:
  test, t1, t2

Examples:
  test this:
  test command

Available Commands:
  sub-test   short sub test description

Options:
    --bflag0=false:
        persistent bool flag with default false value

    --bflag1=true:
        bool with default true value

    --dflag='1m0s':
        duration flag

    --iflag=10:
        integer flag

    --sarray=[a,b]:
        string array flag with default array value

    --sflag='defstr':
        string flag with default value

    -s, --spflag:
        string flag with shorthand

Usage:
  test [flags]
`, sb.String())
	})

	t.Run("test sub command", func(t *testing.T) {
		sb := &strings.Builder{}
		sb.WriteRune('\n')
		subcommand.SetOut(sb)

		FormatHelp(subcommand, nil)
		assert.Equal(t, `
short sub test description

long sub test description
multiline

Options:
    --i64=42:
        int64 flag with default value

Global Options:
    --bflag0=false:
        persistent bool flag with default false value

Usage:
  test sub-test [flags]
`, sb.String())
	})
}
