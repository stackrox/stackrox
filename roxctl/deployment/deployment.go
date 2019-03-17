package deployment

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/deployment/check"
)

// Command defines the image command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "deployment",
		Short: "Deployment defines the actions that you can take against a specific deployment.",
		Long:  "Deployment defines the actions that you can take against a specific deployment.",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
	}

	c.AddCommand(check.Command())
	flags.AddTimeoutWithDefault(c, 1*time.Minute)
	return c
}
