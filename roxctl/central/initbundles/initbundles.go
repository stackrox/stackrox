package initbundles

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/common/environment"
)

// Command defines the bootstrap-token command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "init-bundles",
	}

	c.AddCommand(
		generateCommand(cliEnvironment),
		listCommand(cliEnvironment),
		revokeCommand(cliEnvironment),
		fetchCACommand(cliEnvironment),
	)

	return c
}
