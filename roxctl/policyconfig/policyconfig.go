package policyconfig

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/policyconfig/reconcile"
)

// Command defines the policy-config command tree.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy-config",
		Short: "Commands for managing policies as code",
	}
	cmd.AddCommand(
		reconcile.Command(cliEnvironment),
	)

	flags.HideInheritedFlags(cmd)

	return cmd
}
