package initbundles

import (
	"github.com/spf13/cobra"
)

// Command defines the bootstrap-token command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "init-bundles",
	}

	c.AddCommand(
		generateCommand(),
		listCommand(),
		revokeCommand(),
		fetchCACommand(),
	)

	return c
}
