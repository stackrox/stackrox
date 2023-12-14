package stream

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/stream/alerts"
	"github.com/stackrox/rox/roxctl/stream/deployments"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "stream",
		Short: "Commands related to streaming data from Central.",
	}
	c.AddCommand(
		deployments.Command(cliEnvironment),
		alerts.Command(cliEnvironment),
	)
	return c
}
