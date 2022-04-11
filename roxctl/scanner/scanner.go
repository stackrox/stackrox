package scanner

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/scanner/generate"
	"github.com/stackrox/rox/roxctl/scanner/uploaddb"
)

// Command controls all of the functions being applied to a sensor
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "scanner",
	}
	flags.AddTimeoutWithDefault(c, time.Minute)
	c.AddCommand(
		generate.Command(cliEnvironment),
		uploaddb.Command(cliEnvironment),
	)
	return c
}
