package create

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
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

	c.PersistentFlags().String(k8sobject.ConfigMapFlag, "", `Config Map to which the declarative config YAML should be written to.
If this is unset and the flag "secret" is also unset, the created YAML will be printed to stdout instead`)
	c.PersistentFlags().String(k8sobject.SecretFlag, "", `Secret to which the declarative config YAML should be written to.
Secrets should be used in case sensitive data is contained within the declarative configuration, e.g. within auth providers.
If this is unset and the flag "config-map" is also unset, the created YAML will be printed to stdout instead.`)
	c.PersistentFlags().String(k8sobject.NamespaceFlag, "", `Only required in case the declarative config YAML should be written to a Config Map or secret.
If left empty, the default namespace in the current kube config will be used.`)

	c.MarkFlagsMutuallyExclusive(k8sobject.ConfigMapFlag, k8sobject.SecretFlag)

	return c
}
