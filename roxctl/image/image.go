package image

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/image/check"
)

// Command defines the image command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "image",
		Short: "Image defines the actions that you can take against a specific image.",
		Long:  "Image defines the actions that you can take against a specific image.",
		Run: func(c *cobra.Command, _ []string) {
			c.Help()
		},
	}

	c.AddCommand(check.Command())
	return c
}
