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
	c.Flags().Bool("bflag1", true, "bool with default true value")
	c.Flags().String("sflag", "defstr", "string flag with default value")
	c.Flags().Int("iflag", 10, "integer flag")
	c.Flags().Duration("dflag", time.Minute, "duration flag")
	c.Flags().StringP("spflag", "s", "", "string flag with shorthand")
	c.Flags().StringArray("sarray", []string{"a", "b"}, "string array flag with default array value")

	fs := pflag.NewFlagSet("test", pflag.ExitOnError)
	fs.String("flagset", "", "string flag from a test set")
	c.Flags().AddFlagSet(fs)

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
	subcommand.AddCommand(subsubcommand)

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
  sub-sub-test   short subsub test description

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
}
