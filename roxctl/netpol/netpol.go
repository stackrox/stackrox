package netpol

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/netpol/connectivity"
	"github.com/stackrox/rox/roxctl/netpol/generate"
)

// Command defines the netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "netpol",
		Short: "Commands related to network policies",
		Long:  `Commands related to network policies.`,
	}

	c.AddCommand(
		connectivity.Command(cliEnvironment),
		generate.Command(cliEnvironment),
	)
	return c
}
