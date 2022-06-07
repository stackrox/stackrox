package collector

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages"
	"github.com/stackrox/rox/roxctl/common"
)

// Command defines the collector command tree
func Command(cliEnvironment common.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "collector",
	}

	c.AddCommand(
		supportpackages.Command(cliEnvironment),
	)
	return c
}
