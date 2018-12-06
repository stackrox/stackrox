package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/cmd/deploy/central"
	"github.com/stackrox/rox/generated/api/v1"
)

func volumeCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s adds an external volume to the deployment definition", name),
		Long: fmt.Sprintf(`%s adds an external volume to the deployment definition.
Output is a zip file printed to stdout.`, name),
		Example:       "Enter volume",
		SilenceErrors: true,
		Annotations: map[string]string{
			categoryAnnotation: "Enter volume",
		},
	}
}

func externalVolume() *cobra.Command {
	external := new(central.ExternalPersistence)
	c := volumeCommand("pvc")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.External = external
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}
	c.Flags().StringVarP(&external.Name, "name", "", "stackrox-db", "external volume name")
	c.Flags().StringVarP(&external.StorageClass, "storage-class", "", "", "storage class name (optional)")
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

func hostPathVolume(cluster v1.ClusterType) *cobra.Command {
	hostpath := new(central.HostPathPersistence)
	c := volumeCommand("hostpath")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.HostPath = hostpath
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}
	c.Flags().StringVarP(&hostpath.HostPath, "hostpath", "", "/var/lib/stackrox", "path on the host")

	var defaultSelector string
	switch cluster {
	case v1.ClusterType_SWARM_CLUSTER, v1.ClusterType_DOCKER_EE_CLUSTER:
		defaultSelector = "node.hostname"
	case v1.ClusterType_KUBERNETES_CLUSTER, v1.ClusterType_OPENSHIFT_CLUSTER:
		defaultSelector = "kubernetes.io/hostname"
	}
	c.Flags().StringVarP(&hostpath.NodeSelectorKey, "node-selector-key", "", "", fmt.Sprintf("node selector key (e.g. %s)", defaultSelector))
	c.Flags().StringVarP(&hostpath.NodeSelectorValue, "node-selector-value", "", "", "node selector value")

	return c
}
