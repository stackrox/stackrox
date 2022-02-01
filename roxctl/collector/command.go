package collector

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages"
)

// Command defines the collector command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "collector",
	}

	c.AddCommand(
		supportpackages.Command(),
	)
	return c
}
