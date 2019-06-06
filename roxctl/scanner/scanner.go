package scanner

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/scanner/generate"
)

// Command controls all of the functions being applied to a sensor
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "scanner",
		Short: "Scanner is the list of commands that pertain to the StackRox Scanner",
		Long:  "Scanner is the list of commands that pertain to the StackRox Scanner",
	}
	flags.AddTimeout(c)
	c.AddCommand(generate.Command())
	return c
}
