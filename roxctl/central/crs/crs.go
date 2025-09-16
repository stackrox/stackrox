package crs

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
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

	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)

	return c
}
