package central

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/deploy"
)

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "central",
		Short: "Central is the list of commands that pertain to the Central service",
		Long:  "Central is the list of commands that pertain to the Central service",
		Run: func(c *cobra.Command, _ []string) {
			c.Help()
		},
	}

	c.AddCommand(deploy.Command())
	return c
}
