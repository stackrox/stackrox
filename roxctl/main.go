package main

import (
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/clientconn"
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
			var commands []string
			for c := cmd; c != nil; c = c.Parent() {
				commands = append([]string{c.Name()}, commands...)
				c.Flags().Visit(func(f *pflag.Flag) {
					if f.Changed {
						commands = append(commands, "--"+f.Name)
						if f.Value.Type() == "stringSlice" || f.Value.Type() == "string" {
							commands = append(commands, "...")
						} else {
							commands = append(commands, f.Value.String())
						}
					}
				})
			}
			common.RoxctlCommand = strings.Join(commands[1:], " ")
		})
	})

	clientconn.SetUserAgent(clientconn.Roxctl)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
