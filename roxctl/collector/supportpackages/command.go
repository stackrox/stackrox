package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages/upload"
	"github.com/stackrox/rox/roxctl/common"
)

// Command defines the central command tree
func Command(cliEnvironment common.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "support-packages",
	}

	c.AddCommand(
		upload.Command(cliEnvironment),
	)
	return c
}
