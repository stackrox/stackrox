package netpol

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/netpol/generate"
)

// Command defines the generate netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := generate.NewNetpolGenerateCmd(cliEnvironment)
	c := &cobra.Command{
		Use:   "netpol <folder-path>",
		Short: cmd.ShortText(),
		Long:  cmd.LongText(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cliEnvironment.Logger().WarnfLn("Command 'generate netpol' is deprecated. Use 'netpol generate' instead.")
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(c *cobra.Command, args []string) error {
			//nolint:wrapcheck // this is alias command, so it is not considered as separate pkg
			return cmd.RunE(c, args)
		},
	}
	return cmd.AddFlags(c)
}
