package generate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/roxctl/common/environment"
)

func volumeCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Adds an external %s volume to the deployment definition", name),
		Long: fmt.Sprintf(`Adds an external %s volume to the deployment definition.
Output is a zip file printed to stdout.`, name),
		Annotations: map[string]string{
			categoryAnnotation: "External volume type",
		},
	}
}

func externalVolume(cliEnvironment environment.Environment) *cobra.Command {
	external := &renderer.ExternalPersistence{
		DB: &renderer.ExternalPersistenceInstance{},
	}
	c := volumeCommand("pvc")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.External = external
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return OutputZip(cliEnvironment.Logger(), cliEnvironment.InputOutput(), cfg)
	}
	flagWrap := &flagsWrapper{FlagSet: c.Flags()}
	flagWrap.StringVarP(&external.DB.Name, "db-name", "", "central-db", "External volume name for Central DB.", "central-db")
	flagWrap.StringVarP(&external.DB.StorageClass, "db-storage-class", "", "", "Storage class name for Central DB (optional if you have a default StorageClass configured).", "central-db")
	flagWrap.Uint32VarP(&external.DB.Size, "db-size", "", 100, "External volume size in Gi for Central DB.", "central-db")
	return c
}

func noVolume(cliEnvironment environment.Environment) *cobra.Command {
	c := volumeCommand("none")
	c.RunE = func(c *cobra.Command, args []string) error {
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return OutputZip(cliEnvironment.Logger(), cliEnvironment.InputOutput(), cfg)
	}
	c.Hidden = true
	return c
}

func hostPathVolume(cliEnvironment environment.Environment) *cobra.Command {
	hostpath := &renderer.HostPathPersistence{
		DB: &renderer.HostPathPersistenceInstance{},
	}
	c := volumeCommand("hostpath")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.HostPath = hostpath
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return OutputZip(cliEnvironment.Logger(), cliEnvironment.InputOutput(), cfg)
	}
	c.Flags().StringVarP(&hostpath.DB.HostPath, "db-hostpath", "", "/var/lib/stackrox-central", "Path on the host.")
	c.Flags().StringVarP(&hostpath.DB.NodeSelectorKey, "db-node-selector-key", "", "", "Node selector key (e.g. kubernetes.io/hostname).")
	c.Flags().StringVarP(&hostpath.DB.NodeSelectorValue, "db-node-selector-value", "", "", "Node selector value.")

	return c
}
