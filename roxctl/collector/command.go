package collector

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages"
)

// Command defines the collector command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "collector",
		Short: "collector is the list of commands that pertain to the Collector service",
		Long:  "collector is the list of commands that pertain to the Collector service",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
	}

	c.AddCommand(
		supportpackages.Command(),
	)
	return c
}
