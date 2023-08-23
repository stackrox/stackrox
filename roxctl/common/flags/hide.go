package flags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/set"
)

// HideInheritedFlags hides all inherited flags except the flag names given.
// This is especially useful for commands that do not require central connectivity and want to avoid exposing these
// flags to the client to avoid confusion.
func HideInheritedFlags(cmd *cobra.Command, flagsToShow ...string) {
	flagSet := set.NewStringSet(flagsToShow...)
	orig := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, strings []string) {
		cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
			if !flagSet.Contains(flag.Name) {
				flag.Hidden = true
			}
		})
		orig(cmd, strings)
	})
}
