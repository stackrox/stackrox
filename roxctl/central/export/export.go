package export

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/export/deployments"
	"github.com/stackrox/rox/roxctl/central/export/images"
	"github.com/stackrox/rox/roxctl/central/export/nodes"
	"github.com/stackrox/rox/roxctl/central/export/pods"
	"github.com/stackrox/rox/roxctl/common/environment"
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
