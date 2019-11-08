package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages/upload"
)

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "support-packages",
		Short: "The list of commands that pertain to uploading support-packages for collector",
		Long:  "The list of commands that pertain to uploading support-packages for collector",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
	}

	c.AddCommand(
		upload.Command(),
	)
	return c
}
