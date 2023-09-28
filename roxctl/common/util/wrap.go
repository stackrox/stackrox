package util

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
)

// RunENoArgs is a wrapper for RunE that does not consult the args argument.
func RunENoArgs(f func(*cobra.Command) error) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errox.InvalidArgs.New("expected no arguments; please check usage")
		}
		return f(c)
	}
}

// RunEOneArg is a wrapper for RunE that requires exactly one argument.
func RunEOneArg(f func(*cobra.Command) error) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errox.InvalidArgs.New("expected one argument; please check usage")
		}
		return f(c)
	}
}
