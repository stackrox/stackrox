package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func processFlag(flag *pflag.Flag) {
	const defaultFalse = " (default false)"
	if flag.Value.Type() == "bool" && flag.DefValue == "false" && flag.Name != "help" &&
		!strings.HasSuffix(flag.Usage, defaultFalse) {
		flag.Usage += defaultFalse
	}
}

func processCommandFlags(command *cobra.Command) {
	// LocalFlags() calls mergePersistentFlags() so PersistentFlags are merged
	// to Flags(), but returns only local flags, so PersistentFlags are not
	// revisited on every command.
	command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
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
