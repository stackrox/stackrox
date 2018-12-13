package sensor

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/sensor/generate"
)

// Command controls all of the functions being applied to a sensor
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "sensor",
		Short: "Sensor is the list of commands that pertain to the Sensor service",
		Long:  "Sensor is the list of commands that pertain to the Sensor service",
	}
	c.AddCommand(generate.Command())
	return c
}
