package initbundles

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common"
)

// Command defines the bootstrap-token command tree
func Command(cliEnvironment common.Environment) *cobra.Command {
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
