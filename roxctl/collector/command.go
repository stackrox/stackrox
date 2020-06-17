package collector

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages"
)

// Command defines the collector command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "collector",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
	}

	c.AddCommand(
		supportpackages.Command(),
	)
	return c
}
