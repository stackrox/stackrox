package helm

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/roxctl/helm/derivelocalvalues"
	"github.com/stackrox/rox/roxctl/helm/output"
)

// Command defines the helm command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:    "helm",
		Hidden: !features.CentralInstallationExperience.Enabled(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !features.CentralInstallationExperience.Enabled() {
				fmt.Fprintln(os.Stderr, "Experimental command 'helm' unavailable")
				return errors.New("command unavailable")
			}
			return nil
		},
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
	}

	c.AddCommand(output.Command())
	c.AddCommand(derivelocalvalues.Command())

	return c
}
