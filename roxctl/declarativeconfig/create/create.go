package create

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command defines the declarative config create command tree.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "create",
	}

	c.AddCommand(
		accessScopeCommand(cliEnvironment),
		authProviderCommand(cliEnvironment),
		permissionSetCommand(cliEnvironment),
		roleCommand(cliEnvironment),
	)
	return c
}
