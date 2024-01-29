package sensor

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/sensor/generate"
	"github.com/stackrox/rox/roxctl/sensor/generatecerts"
	"github.com/stackrox/rox/roxctl/sensor/getbundle"
)

// Command controls all of the functions being applied to a sensor
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "sensor",
		Short: "Commands related to deploying StackRox services in secured clusters.",
	}
	c.AddCommand(
		generate.Command(cliEnvironment),
		getbundle.Command(cliEnvironment),
		generatecerts.Command(cliEnvironment),
	)
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}
