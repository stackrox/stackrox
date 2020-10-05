package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages/upload"
)

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "support-packages",
	}

	c.AddCommand(
		upload.Command(),
	)
	return c
}
