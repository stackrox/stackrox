package util

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunENoArgs is a wrapper for RunE that does not consult the args argument.
func RunENoArgs(f func(*cobra.Command) error) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("expected no arguments; please check usage")
		}
		return f(c)
	}
}
