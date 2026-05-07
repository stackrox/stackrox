package migratetooperator

import (
	"github.com/spf13/cobra"

	"github.com/stackrox/rox/pkg/migratetooperator"
	"github.com/stackrox/rox/roxctl/common/environment"
	commonMigrate "github.com/stackrox/rox/roxctl/common/migratetooperator"
	"github.com/stackrox/rox/roxctl/common/util"
)

// Command defines the sensor migrate-to-operator command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &commonMigrate.Command{Env: cliEnvironment}
	c := &cobra.Command{
		Use:   "migrate-to-operator",
		Short: "Generate a SecuredCluster custom resource from existing sensor manifests",
		Long: `Inspects an existing StackRox Sensor deployment (from a directory of manifests
or a live cluster) and produces a SecuredCluster custom resource YAML that
preserves the detected configuration, allowing the StackRox operator to
seamlessly take over management of the deployment.`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return commonMigrate.Run(cmd, migratetooperator.TransformToSecuredCluster)
		}),
	}
	cmd.AddFlags(c)
	return c
}
