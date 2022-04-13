package collector

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/collector/supportpackages"
	"github.com/stackrox/stackrox/roxctl/common/environment"
)

// Command defines the collector command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "collector",
	}

	c.AddCommand(
		supportpackages.Command(cliEnvironment),
	)
	return c
}
