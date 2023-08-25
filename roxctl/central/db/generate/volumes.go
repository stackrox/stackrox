package generate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/roxctl/common/environment"
)

const (
	defaultHostPathPath = "/var/lib/stackrox-central-db"
)

func volumeCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("adds a %s", name),
		Long:  fmt.Sprintf(`adds a %s external volume to Central DB`, name),
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
		return outputZip(cliEnvironment.Logger(), cfg)
	}
	c.Flags().StringVarP(&external.DB.Name, "name", "", "central-db", "external volume name for Central DB")
	c.Flags().StringVarP(&external.DB.StorageClass, "storage-class", "", "", "storage class name for Central DB (optional if you have a default StorageClass configured)")
	c.Flags().Uint32VarP(&external.DB.Size, "size", "", 100, "external volume size in Gi for Central DB")
	return c
}

func noVolume(cliEnvironment environment.Environment) *cobra.Command {
	c := volumeCommand("none")
	c.RunE = func(c *cobra.Command, args []string) error {
		if err := validateConfig(&cfg); err != nil {
			return err
		}
		return outputZip(cliEnvironment.Logger(), cfg)
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
		return outputZip(cliEnvironment.Logger(), cfg)
	}
	c.Flags().StringVarP(&hostpath.DB.HostPath, "hostpath", "", defaultHostPathPath, "path on the host")
	c.Flags().StringVarP(&hostpath.DB.NodeSelectorKey, "node-selector-key", "", "", "node selector key (e.g. kubernetes.io/hostname)")
	c.Flags().StringVarP(&hostpath.DB.NodeSelectorValue, "node-selector-value", "", "", "node selector value")

	return c
}
