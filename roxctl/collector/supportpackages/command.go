package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages/upload"
	"github.com/stackrox/rox/roxctl/common/environment"
)

const (
	supportPkgHelpLong = `Commands to upload support packages for Collector.
Note: uploaded support packages will only affect Secured Clusters on versions
less than 4.5. Newer versions do not require support packages.`
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "support-packages",
		Short: "Commands to upload support packages for Collector",
		Long:  supportPkgHelpLong,
	}

	c.AddCommand(
		upload.Command(cliEnvironment),
	)
	return c
}
