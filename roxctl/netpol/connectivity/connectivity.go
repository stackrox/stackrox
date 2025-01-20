package connectivity

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/netpol/connectivity/diff"
	connectivitymap "github.com/stackrox/rox/roxctl/netpol/connectivity/map"
)

// Command defines the netpol connectivity command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "connectivity",
		Short: "Commands related to connectivity analysis of network policy resources",
		Long:  `Commands related to connectivity analysis of network policy resources.`,
	}

	c.AddCommand(
		connectivitymap.Command(cliEnvironment),
		diff.Command(cliEnvironment),
	)
	return c
}
