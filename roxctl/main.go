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
