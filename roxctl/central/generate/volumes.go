package generate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/roxctl/common/environment"
)

func volumeCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s adds an external volume to the deployment definition", name),
		Long: fmt.Sprintf(`%s adds an external volume to the deployment definition.
Output is a zip file printed to stdout.`, name),
		Example: "Enter Central volume type",
		Annotations: map[string]string{
			categoryAnnotation: "Enter Central volume type",
		},
	}
}

func externalVolume(cliEnvironment environment.Environment) *cobra.Command {
	external := &renderer.ExternalPersistence{
		Central: &renderer.ExternalPersistenceInstance{},
	}
	if features.PostgresDatastore.Enabled() {
		external.DB = &renderer.ExternalPersistenceInstance{}
	}
	c := volumeCommand("pvc")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.External = external
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return OutputZip(cliEnvironment.Logger(), cfg)
	}
	c.Flags().StringVarP(&external.Central.Name, "name", "", "stackrox-db", "external volume name")
	c.Flags().StringVarP(&external.Central.StorageClass, "storage-class", "", "", "storage class name (optional if you have a default StorageClass configured)")
	c.Flags().Uint32VarP(&external.Central.Size, "size", "", 100, "external volume size in Gi")
	if features.PostgresDatastore.Enabled() {
		c.Flags().StringVarP(&external.DB.Name, "db-name", "", "central-db", "external volume name")
		c.Flags().StringVarP(&external.DB.StorageClass, "db-storage-class", "", "", "storage class name (optional if you have a default StorageClass configured)")
		c.Flags().Uint32VarP(&external.DB.Size, "db-size", "", 100, "external volume size in Gi")
	}
	return c
}

func noVolume(cliEnvironment environment.Environment) *cobra.Command {
	c := volumeCommand("none")
	c.RunE = func(c *cobra.Command, args []string) error {
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return OutputZip(cliEnvironment.Logger(), cfg)
	}
	c.Hidden = true
	return c
}

func hostPathVolume(cliEnvironment environment.Environment) *cobra.Command {
	hostpath := &renderer.HostPathPersistence{
		Central: &renderer.HostPathPersistenceInstance{},
	}
	if features.PostgresDatastore.Enabled() {
		hostpath.DB = &renderer.HostPathPersistenceInstance{}
	}
	c := volumeCommand("hostpath")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.HostPath = hostpath
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return OutputZip(cliEnvironment.Logger(), cfg)
	}
	c.Flags().StringVarP(&hostpath.Central.HostPath, "hostpath", "", "/var/lib/stackrox", "path on the host")
	c.Flags().StringVarP(&hostpath.Central.NodeSelectorKey, "node-selector-key", "", "", "node selector key (e.g. kubernetes.io/hostname)")
	c.Flags().StringVarP(&hostpath.Central.NodeSelectorValue, "node-selector-value", "", "", "node selector value")
	if features.PostgresDatastore.Enabled() {
		c.Flags().StringVarP(&hostpath.DB.HostPath, "db-hostpath", "", "/var/lib/stackrox-central", "path on the host")
		c.Flags().StringVarP(&hostpath.DB.NodeSelectorKey, "db-node-selector-key", "", "", "node selector key (e.g. kubernetes.io/hostname)")
		c.Flags().StringVarP(&hostpath.DB.NodeSelectorValue, "db-node-selector-value", "", "", "node selector value")
	}

	return c
}
