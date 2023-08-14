package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/generate/netpol"
)

// Command defines the generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:        "generate",
		Deprecated: "for 'roxctl generate netpol' use 'roxctl netpol generate' instead",
		Short:      "(Technology Preview) Commands related to generating different resources.",
		Long: `Commands related to generating different resources.

** This is a Technology Preview feature **
Technology Preview features are not supported with Red Hat production service level agreements (SLAs) and might not be functionally complete.
Red Hat does not recommend using them in production.
These features provide early access to upcoming product features, enabling customers to test functionality and provide feedback during the development process.
For more information about the support scope of Red Hat Technology Preview features, see https://access.redhat.com/support/offerings/techpreview/`,
	}

	c.AddCommand(
		netpol.Command(cliEnvironment),
	)
	return c
}
