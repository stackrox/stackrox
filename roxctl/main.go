package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/maincommand"
	"github.com/stackrox/stackrox/roxctl/properties"

	// Make sure devbuild setting is registered.
	_ "github.com/stackrox/stackrox/pkg/devbuild"
)

func main() {
	c := maincommand.Command()
	addHelp(c.Commands())

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
		setDescription(c)
		addHelp(c.Commands())
	}
}

// setDescription sets the description for the cobra command in the form of Short and Long.
// If Short / Long is already set on the command, they will take precedence over the properties in the properties file.
// If no Short / Long is set, the properties file will be used to determine both values.
// The function will panic if Short could not be set and is empty.
func setDescription(c *cobra.Command) {
	if c.Short == "" {
		c.Short = properties.MustGetProperty(properties.GetShortCommandKey(c.CommandPath()))
	}
	if c.Long == "" {
		// We do not need to panic here, Long can be empty for commands and is not required like Short.
		c.Long = properties.GetProperty(properties.GetLongCommandKey(c.CommandPath()))
	}
}
