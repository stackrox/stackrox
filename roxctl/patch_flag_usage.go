package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func processFlag(flag *pflag.Flag) {
	if flag.Value.Type() == "bool" && flag.DefValue == "false" && flag.Name != "help" {
		const defaultFalseSuffix = " (default false)"
		if !strings.HasSuffix(flag.Usage, defaultFalseSuffix) {
			flag.Usage += defaultFalseSuffix
		}
	}
}

func processCommandFlags(command *cobra.Command) {
	// LocalFlags() calls mergePersistentFlags() so PersistentFlags are merged.
	_ = command.LocalFlags()
	command.Flags().VisitAll(func(flag *pflag.Flag) {
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
