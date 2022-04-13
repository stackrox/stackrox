package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/collector/supportpackages/upload"
	"github.com/stackrox/stackrox/roxctl/common/environment"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "support-packages",
	}

	c.AddCommand(
		upload.Command(cliEnvironment),
	)
	return c
}
