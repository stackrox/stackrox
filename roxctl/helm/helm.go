package helm

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/roxctl/common/environment"
	"github.com/stackrox/stackrox/roxctl/helm/derivelocalvalues"
	"github.com/stackrox/stackrox/roxctl/helm/output"
)

// Command defines the helm command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "helm",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	c.AddCommand(output.Command(cliEnvironment))
	c.AddCommand(derivelocalvalues.Command(cliEnvironment))

	return c
}
