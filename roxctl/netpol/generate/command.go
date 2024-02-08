package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the netpol generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &NetpolGenerateCmd{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "generate <folder-path>",
		Short: cmd.ShortText(),
		Long:  cmd.LongText(),
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.RunE(c, args)
		},
	}
	return cmd.AddFlags(c)
}
