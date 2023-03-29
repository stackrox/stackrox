package collector

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the collector command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "collector",
		Short: "Commands related to the Collector service.",
	}

	c.AddCommand(
		supportpackages.Command(cliEnvironment),
	)
	return c
}
