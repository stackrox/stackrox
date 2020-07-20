package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stackrox/rox/roxctl/properties"

	// Make sure devbuild setting is registered.
	_ "github.com/stackrox/rox/pkg/devbuild"
)

func main() {
	c := maincommand.Command()
	addHelp(c.Commands())

	flags.AddPassword(c)
	flags.AddConnectionFlags(c)
	flags.AddAPITokenFile(c)

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

	PatchPersistentPreRunHooks(c)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

func addHelp(commands []*cobra.Command) {
	if len(commands) == 0 {
		return
	}
	for _, c := range commands {
		c.Short = properties.MustGetProperty(properties.GetShortCommandKey(c.CommandPath()))
		c.Long = properties.MustGetProperty(properties.GetLongCommandKey(c.CommandPath()))
		addHelp(c.Commands())
	}
}
