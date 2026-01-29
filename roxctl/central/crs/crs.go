package crs

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the bootstrap-token command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "crs",
		Short: "Manage Cluster Registration Secrets (CRSs)",
	}

	c.AddCommand(
		generateCommand(cliEnvironment),
		listCommand(cliEnvironment),
		revokeCommand(cliEnvironment),
	)

	return c
}
