package sensor

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/sensor/generate"
	"github.com/stackrox/rox/roxctl/sensor/getbundle"
)

// Command controls all of the functions being applied to a sensor
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "sensor",
		Short: "The list of commands that pertain to the Sensor service",
		Long:  "The list of commands that pertain to the Sensor service",
	}
	c.AddCommand(
		generate.Command(),
		getbundle.Command(),
	)
	flags.AddTimeout(c)
	return c
}
