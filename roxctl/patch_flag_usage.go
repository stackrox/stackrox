package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func processFlag(flag *pflag.Flag) {
	if flag.Value.Type() == "bool" && flag.DefValue == "false" {
		flag.Usage = fmt.Sprintf("%s (default %s)", flag.Usage, flag.DefValue)
	}
}

func processCommandFlags(command *cobra.Command) {
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		processFlag(flag)
	})
	command.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		processFlag(flag)
	})
}

// AddMissingDefaultsToFlagUsage processes the tree of commands starting at the provided command
// and adds default values to flag usage information where necessary (i.e. boolean flags defaulting to `false`).
func AddMissingDefaultsToFlagUsage(command *cobra.Command) {
	processCommandFlags(command)
	for _, subcommand := range command.Commands() {
		AddMissingDefaultsToFlagUsage(subcommand)
	}
}
