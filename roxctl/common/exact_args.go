package common

import (
	"errors"

	"github.com/spf13/cobra"
)

// ExactArgsWithCustomErrMessage returns an error with a custom message
// if there are not exactly n args
func ExactArgsWithCustomErrMessage(n int, msg string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return errors.New(msg)
		}
		return nil
	}
}
