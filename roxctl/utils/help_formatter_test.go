package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestFormatHelp(t *testing.T) {
	c := &cobra.Command{
		Use:     "test",
		Short:   "short description",
		Long:    "long description\nmultiline",
		Aliases: []string{"t1", "t2"},
		Example: "test this:\ntest command",
	}
	c.PersistentFlags().Bool("bflag0", false, "persistent bool flag with default false value")
	c.LocalFlags().Bool("bflag1", true, "bool with default true value")
	c.LocalFlags().String("sflag", "defstr", "string flag with default value")
	c.LocalFlags().Int("iflag", 10, "integer flag")
	c.LocalFlags().Duration("dflag", time.Minute, "duration flag")
	c.LocalFlags().StringP("spflag", "s", "", "string flag with shorthand")
	c.LocalFlags().StringArray("sarray", []string{"a", "b"}, "string array flag with default array value")

	fs := pflag.NewFlagSet("test", pflag.ExitOnError)
	fs.String("flagset", "", "string flag from a test set")
	c.LocalFlags().AddFlagSet(fs)

	c.LocalFlags().String("depreflag", "deprecated", "string deprecated flag with default value")
	assert.NoError(t, c.LocalFlags().MarkDeprecated("depreflag", "don't use"))

	subcommand := &cobra.Command{
		Use:   "sub-test",
		Short: "short sub test description",
		Long:  "long sub test description\nmultiline",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	subcommand.Flags().Int64("i64", int64(42), "int64 flag with default value")

	subcommand.AddGroup(&cobra.Group{
		ID:    "g1",
		Title: "Test Group",
	})
	subsubcommand := &cobra.Command{
		Use:     "sub-sub-test",
		Short:   "short subsub test description",
		Run:     func(cmd *cobra.Command, args []string) {},
		GroupID: "g1",
	}
	subsubcommand1 := &cobra.Command{
		Use:   "sub-sub-test1",
		Short: "short subsub test1 description",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	subcommand.AddCommand(subsubcommand, subsubcommand1)

	deprecatedCommand := &cobra.Command{
		Use:        "deprecated-test",
		Short:      "short depreacted command test description",
		Long:       "long depreacted command test description",
		Run:        func(cmd *cobra.Command, args []string) {},
		Deprecated: "don't use",
	}

	c.AddCommand(subcommand, deprecatedCommand)

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

    --flagset:
        string flag from a test set

    --iflag=10:
        integer flag

    --sarray=[a,b]:
        string array flag with default array value

    --sflag='defstr':
        string flag with default value

    -s, --spflag:
        string flag with shorthand

Usage:
  test [command]
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

Test Group
  sub-sub-test    short subsub test description

Additional Commands:
  sub-sub-test1   short subsub test1 description

Options:
    --i64=42:
        int64 flag with default value

Global Options:
    --bflag0=false:
        persistent bool flag with default false value

Usage:
  test sub-test [flags]
  test sub-test [command]
`, sb.String())
	})

	t.Run("test deprecated command", func(t *testing.T) {
		sb := &strings.Builder{}
		sb.WriteRune('\n')
		deprecatedCommand.SetOut(sb)

		FormatHelp(deprecatedCommand, nil)
		assert.Equal(t, `
short depreacted command test description

long depreacted command test description

Global Options:
    --bflag0=false:
        persistent bool flag with default false value

Usage:
  test deprecated-test [flags]

DEPRECATED: don't use
`, sb.String())
	})

	t.Run("don't write after errors", func(t *testing.T) {
		sb := &sbWithErrors{fail: func(_ int, s string) bool {
			return s == "X"
		}}
		w := makeHelpWriter(
			makeFormattingWriter(sb, 40, defaultTabWidth))

		w.Write("A")
		w.Write("B", "X <- bad token")
		w.WriteLn("C", "D")

		assert.ErrorIs(t, w.err, errBadToken)
		assert.Equal(t, "AB", sb.String())
	})
}
