package initbundles

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/features"
)

// Command defines the bootstrap-token command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:    "init-bundles",
		Hidden: !features.SensorInstallationExperience.Enabled(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !features.SensorInstallationExperience.Enabled() {
				fmt.Fprintln(os.Stderr, "Experimental command 'init-bundles' unavailable")
				return errors.New("command unavailable")
			}
			return nil
		},
	}

	c.AddCommand(
		generateCommand(),
		listCommand(),
		revokeCommand(),
		fetchCACommand(),
	)

	return c
}
