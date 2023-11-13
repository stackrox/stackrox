package cluster

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/cluster/delete"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a sensor
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "cluster",
		Short: "Commands related to a cluster.",
	}

	c.AddCommand(delete.Command(cliEnvironment))
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}
