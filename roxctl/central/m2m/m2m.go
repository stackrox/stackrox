package m2m

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/m2m/exchange"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command adds the m2m command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:     "machine-to-machine",
		Aliases: []string{"m2m"},
		Short:   "Commands for managing machine to machine authentication",
	}

	c.AddCommand(exchange.Command(cliEnvironment))
	return c
}
