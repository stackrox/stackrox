package get

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	cluster "github.com/stackrox/rox/roxctl/generated/get/cluster"
	policy "github.com/stackrox/rox/roxctl/generated/get/policy"
	policymitrevectors "github.com/stackrox/rox/roxctl/generated/get/policymitrevectors"
)

// Command defines the get command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Display resources",
	}
	c.AddCommand(cluster.Command(cliEnvironment))
	c.AddCommand(policy.Command(cliEnvironment))
	c.AddCommand(policymitrevectors.Command(cliEnvironment))

	flags.AddTimeoutWithDefault(c, time.Minute)
	return c
}
