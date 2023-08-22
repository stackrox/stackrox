package connectivity_map

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	connectivitymap "github.com/stackrox/rox/roxctl/netpol/connectivity/map"
)

// Command defines the connectivity-map command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := connectivitymap.NewCmd(cliEnvironment)
	c := &cobra.Command{
		Use:   "connectivity-map <folder-path>",
		Short: "(Technology Preview) Analyze connectivity based on network policies and other resources.",
		Long: `Based on a given folder containing deployment and network policy YAMLs, will analyze permitted cluster connectivity. Will write to stdout if no output flags are provided.

** This is a Technology Preview feature **
Technology Preview features are not supported with Red Hat production service level agreements (SLAs) and might not be functionally complete.
Red Hat does not recommend using them in production.
These features provide early access to upcoming product features, enabling customers to test functionality and provide feedback during the development process.
For more information about the support scope of Red Hat Technology Preview features, see https://access.redhat.com/support/offerings/techpreview/`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			cliEnvironment.Logger().WarnfLn("Command 'connectivity-map' is deprecated. Use 'netpol connectivity map' instead.")
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.RunE(c, args)
		},
	}
	return cmd.AddFlags(c)
}
