package declarativeconfig

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/declarativeconfig/create"
	"github.com/stackrox/rox/roxctl/declarativeconfig/lint"
)

// Command defines the declarative config command tree.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "declarative-config",
		Short: "Commands that help manage declarative configuration",
	}
	cmd.AddCommand(
		create.Command(cliEnvironment),
		lint.Command(cliEnvironment),
	)

	flags.HideInheritedFlags(cmd)

	return cmd
}
