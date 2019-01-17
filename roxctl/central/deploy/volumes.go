package deploy

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/deploy/renderer"
)

func volumeCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s adds an external volume to the deployment definition", name),
		Long: fmt.Sprintf(`%s adds an external volume to the deployment definition.
Output is a zip file printed to stdout.`, name),
		Example: "Enter volume",
		Annotations: map[string]string{
			categoryAnnotation: "Enter volume",
		},
	}
}

func externalVolume() *cobra.Command {
	external := new(renderer.ExternalPersistence)
	c := volumeCommand("pvc")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.External = external
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}
	c.Flags().StringVarP(&external.Name, "name", "", "stackrox-db", "external volume name")
	c.Flags().StringVarP(&external.StorageClass, "storage-class", "", "", "storage class name (optional if you have a default StorageClass configured)")
	return c
}

func noVolume() *cobra.Command {
	c := volumeCommand("none")
	c.RunE = func(c *cobra.Command, args []string) error {
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}
	return c
}

func hostPathVolume() *cobra.Command {
	hostpath := new(renderer.HostPathPersistence)
	c := volumeCommand("hostpath")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.HostPath = hostpath
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}
	c.Flags().StringVarP(&hostpath.HostPath, "hostpath", "", "/var/lib/stackrox", "path on the host")
	c.Flags().StringVarP(&hostpath.NodeSelectorKey, "node-selector-key", "", "", "node selector key (e.g. kubernetes.io/hostname)")
	c.Flags().StringVarP(&hostpath.NodeSelectorValue, "node-selector-value", "", "", "node selector value")

	return c
}
