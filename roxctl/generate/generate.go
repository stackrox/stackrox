package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/generate/netpol"
)

// Command defines the generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "(Technology Preview) Commands related to generating different resources.",
		Long:  `Commands related to generating different resources.`,
	}

	c.AddCommand(
		netpol.Command(cliEnvironment),
	)
	return c
}
