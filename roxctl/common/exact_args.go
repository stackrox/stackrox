package common

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
)

// ExactArgsWithCustomErrMessage returns an error with a custom message
// if there are not exactly n args
func ExactArgsWithCustomErrMessage(n int, msg string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return errox.InvalidArgs.New(msg)
		}
		return nil
	}
}
