package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/central"
)

func volumeCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s adds an external volume to the deployment definition", name),
		Long: fmt.Sprintf(`%s adds an external volume to the deployment definition.
Output is a zip file printed to stdout.`, name),
		Example:       "Enter volume (optional)",
		SilenceErrors: true,
		Annotations: map[string]string{
			"category": "Enter volume (optional)",
		},
	}
}

func externalVolume(cluster v1.ClusterType) *cobra.Command {
	external := new(central.ExternalPersistence)
	c := volumeCommand("external")
	c.RunE = func(c *cobra.Command, args []string) error {
		cfg.External = external
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}
	c.Flags().StringVarP(&external.Name, "name", "", "prevent-db", "external volume name")
	c.Flags().StringVarP(&external.MountPath, "mount-path", "", "/var/lib/prevent", "mount path inside the container")
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
	c.Flags().StringVarP(&hostpath.Name, "name", "", "prevent-db", "hostpath volume name")
	c.Flags().StringVarP(&hostpath.HostPath, "hostpath", "", "/var/lib/prevent", "path on the host")
	c.Flags().StringVarP(&hostpath.MountPath, "mount-path", "", "/var/lib/prevent", "mount path inside the container")

	var defaultSelector string
	switch cluster {
	case v1.ClusterType_SWARM_CLUSTER, v1.ClusterType_DOCKER_EE_CLUSTER:
		defaultSelector = "node.hostname"
	case v1.ClusterType_KUBERNETES_CLUSTER, v1.ClusterType_OPENSHIFT_CLUSTER:
		defaultSelector = "kubernetes.io/hostname"
	}
	c.Flags().StringVarP(&hostpath.NodeSelectorKey, "node-selector-key", "", defaultSelector, "node selector key")
	c.Flags().StringVarP(&hostpath.NodeSelectorValue, "node-selector-value", "", "", "node selector value")

	return c
}
