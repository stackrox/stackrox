package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the netpol generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &netpolGenerateCmd{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "generate <folder-path>",
		Short: "Recommend Network Policies based on deployment information",
		Long:  "Based on a given folder containing deployment YAMLs, will generate a list of recommended Network Policies. Will write to stdout if no output flags are provided.",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.RunE(c, args)
		},
	}
	return cmd.AddFlags(c)
}
