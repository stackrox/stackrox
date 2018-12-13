package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/central"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/image"
	"github.com/stackrox/rox/roxctl/sensor"
)

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version of the CLI",
		Run: func(*cobra.Command, []string) {
			fmt.Println(version.GetMainVersion())
		},
	}
}

func main() {
	c := &cobra.Command{
		Use:   "roxctl",
		Short: "roxctl is the command-line interface for StackRox services",
		Long:  "roxctl is the command-line interface for StackRox services",
	}
	c.AddCommand(versionCommand())

	// Image Commands
	c.AddCommand(image.Command())

	// Central Commands
	c.AddCommand(central.Command())

	// Sensor Commands
	c.AddCommand(sensor.Command())

	common.AddAuthFlags(c)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
