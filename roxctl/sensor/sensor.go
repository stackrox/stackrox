package sensor

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/common/environment"
	"github.com/stackrox/stackrox/roxctl/common/flags"
	"github.com/stackrox/stackrox/roxctl/sensor/generate"
	"github.com/stackrox/stackrox/roxctl/sensor/generatecerts"
	"github.com/stackrox/stackrox/roxctl/sensor/getbundle"
)

// Command controls all of the functions being applied to a sensor
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "sensor",
	}
	c.AddCommand(
		generate.Command(cliEnvironment),
		getbundle.Command(cliEnvironment),
		generatecerts.Command(cliEnvironment),
	)
	flags.AddTimeoutWithDefault(c, 30*time.Second)
	return c
}
