package policy

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/policy/upload"
)

// Command defines the policy command tree.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Commands that help manage policies",
	}
	cmd.AddCommand(upload.Command(cliEnvironment))

	flags.HideInheritedFlags(cmd)

	return cmd
}
