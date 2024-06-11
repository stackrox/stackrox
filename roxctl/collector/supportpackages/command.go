package supportpackages

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/collector/supportpackages/upload"
	"github.com/stackrox/rox/roxctl/common/environment"
)

const (
	warningDeprecatedSupportPkg = `WARNING: support-packages has been deprecated
and will be removed in a future release`
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:        "support-packages",
		Short:      "Commands to upload support packages for Collector.",
		Deprecated: warningDeprecatedSupportPkg,
	}

	c.AddCommand(
		upload.Command(cliEnvironment),
	)
	return c
}
