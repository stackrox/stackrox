package helm

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/helm/derivelocalvalues"
	"github.com/stackrox/rox/roxctl/helm/output"
)

// Command defines the helm command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "helm",
		Short: "Commands related to StackRox Helm Charts.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	c.AddCommand(output.Command(cliEnvironment))
	c.AddCommand(derivelocalvalues.Command(cliEnvironment))

	return c
}
