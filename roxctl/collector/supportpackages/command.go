package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages/upload"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "support-packages",
		Short: "Commands to upload support packages for Collector.",
	}

	c.AddCommand(
		upload.Command(cliEnvironment),
	)
	return c
}
