package deployment

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/deployment/check"
)

// Command defines the image command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "deployment",
		Short: "Commands related to deployments.",
	}

	c.AddCommand(check.Command(cliEnvironment))
	// For deployments with unscanned images, the result of the detection service will take longer since scans are
	// being performed in-line. Hence, need to set the timeout to a more generous value.
	// In reality, if the image is already scanned and no force flag is given, this shouldn't take longer than the
	// default timeout.
	flags.AddTimeoutWithDefault(c, 10*time.Minute)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))
	return c
}
