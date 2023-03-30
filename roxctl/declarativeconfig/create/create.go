package create

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/configmap"
)

// Command defines the declarative config create command tree.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Commands related to creating declarative configurations",
	}

	c.AddCommand(
		accessScopeCommand(cliEnvironment),
		authProviderCommand(cliEnvironment),
		permissionSetCommand(cliEnvironment),
		roleCommand(cliEnvironment),
	)

	c.PersistentFlags().String(configmap.ConfigMapFlag, "", `Config Map to which the declarative config YAML should be written to.
If left empty, the created YAML will be printed to stdout instead`)
	c.PersistentFlags().String(configmap.NamespaceFlag, "", `Only required in case the declarative config YAML should be written to a Config Map.
If left empty, the default namespace in the current kube config will be used.`)

	return c
}
