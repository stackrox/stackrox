package connectivity

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	connectivitymap "github.com/stackrox/rox/roxctl/netpol/connectivity/map"
)

// Command defines the netpol connectivity command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "connectivity",
		Short: "(Technology Preview) Commands related to connectivity analysis of network policy resources.",
		Long: `Commands related to connectivity analysis of network policy resources.
** This is a Technology Preview feature **
Technology Preview features are not supported with Red Hat production service level agreements (SLAs) and might not be functionally complete.
Red Hat does not recommend using them in production.
These features provide early access to upcoming product features, enabling customers to test functionality and provide feedback during the development process.
For more information about the support scope of Red Hat Technology Preview features, see https://access.redhat.com/support/offerings/techpreview/`,
	}

	c.AddCommand(
		connectivitymap.Command(cliEnvironment),
	)
	return c
}
