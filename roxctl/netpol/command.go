package netpol

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/netpol/generate"
)

// Command defines the collector command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "netpol",
	}

	c.AddCommand(
		generate.Command(cliEnvironment),
	)
	return c
}
