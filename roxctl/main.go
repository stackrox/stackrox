package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/maincommand"

	// Make sure devbuild setting is registered.
	_ "github.com/stackrox/rox/pkg/devbuild"
)

func main() {
	c := maincommand.Command()

	// This is a workaround. Cobra/pflag takes care of presenting flag usage information
	// to the user including the respective flag default values.
	//
	// But, as an exception, showing the default value for a flag is skipped in pflag if
	// that value is the zero value for a certain standard type.
	//
	// In our case this caused the unintended behaviour of not showing the default values
	// for our boolean flags which default to `false`.
	//
	// Until we have a better solution (e.g. way to control this behaviour in upstream pflag)
	// we simply add the usage information "(default false)" to our affected boolean flags.
	AddMissingDefaultsToFlagUsage(c)

	once := sync.Once{}
	common.PatchPersistentPreRunHooks(c, func(cmd *cobra.Command, args []string) {
		once.Do(func() {
			common.RoxctlCommand = getCommandPath(cmd)
			_ = reconstructArguments(cmd) // Ignore arguments for now (TODO).
		})
	})

	clientconn.SetUserAgent(clientconn.Roxctl)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

func getCommandPath(cmd *cobra.Command) string {
	binaryName := cmd.Root().CommandPath() + " "
	return strings.TrimPrefix(cmd.CommandPath(), binaryName)
}

func reconstructArguments(cmd *cobra.Command) string {
	var arguments []string
	// Reconstruct provided arguments, masking string values:
	for c := cmd; c != nil; c = c.Parent() {
		c.Flags().Visit(func(f *pflag.Flag) {
			arguments = append(arguments, "--"+f.Name)
			if f.Value.Type() == "stringSlice" || f.Value.Type() == "string" {
				arguments = append(arguments, "***")
			} else {
				arguments = append(arguments, f.Value.String())
			}
		})
	}
	// Attention: cmd.Flags().Args() are not included, as may contain sensitive
	// data.

	return strings.Join(arguments, " ")
}
