package export

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/export/deployments"
	"github.com/stackrox/rox/roxctl/export/images"
	"github.com/stackrox/rox/roxctl/export/nodes"
	"github.com/stackrox/rox/roxctl/export/pods"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "export",
		Short: "Commands related to exporting data from Central.",
	}
	c.AddCommand(
		deployments.Command(cliEnvironment),
		images.Command(cliEnvironment),
		nodes.Command(cliEnvironment),
		pods.Command(cliEnvironment),
	)
	return c
}
