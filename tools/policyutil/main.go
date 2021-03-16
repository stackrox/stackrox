package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/tools/policyutil/command"
	"github.com/stackrox/rox/tools/policyutil/common"
)

func main() {
	c := &cobra.Command{
		Use:          fmt.Sprintf("%s <command> [--verbose] ...", os.Args[0]),
		Short:        "StackRox policy utility tool",
		SilenceUsage: true,
	}
	c.PersistentFlags().BoolVarP(
		&common.Verbose,
		common.VerboseFlagName,
		common.VerboseFlagShorthand,
		false,
		"verbose output")
	c.PersistentFlags().BoolVarP(
		&common.Interactive,
		common.InteractiveFlagName,
		common.InteractiveFlagShorthand,
		false,
		"interactive mode")

	c.AddCommand(
		versionCommand(),
		command.Command(),
	)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print current (latest) policy version",
		Run: func(c *cobra.Command, _ []string) {
			common.PrintResult("Current policy version: %q", policyversion.CurrentVersion().String())
		},
	}
}
