package cluster

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/cluster/delete"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a sensor
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "cluster",
	}

	c.AddCommand(delete.Command(cliEnvironment))
	flags.AddTimeoutWithDefault(c, 5*time.Second)
	return c
}
