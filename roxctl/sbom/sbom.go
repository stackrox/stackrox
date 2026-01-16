package sbom

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/sbom/scan"
)

// Command defines the image command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "sbom",
		Short: "Commands that you can run on SBOMs",
	}

	c.AddCommand(scan.Command(cliEnvironment))

	flags.AddTimeoutWithDefault(c, 5*time.Minute)
	return c
}
