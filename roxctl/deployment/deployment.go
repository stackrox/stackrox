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
		Use: "deployment",
	}

	c.AddCommand(check.Command(cliEnvironment))
	flags.AddTimeoutWithDefault(c, 1*time.Minute)
	return c
}
