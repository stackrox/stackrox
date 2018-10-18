package main

import (
	"fmt"

	"github.com/stackrox/rox/cmd/deploy/central"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	cfg central.Config
)

func validateConfig(c central.Config) error {
	if err := validateExternal(c.External, c.ClusterType); err != nil {
		return err
	}
	return validateHostPath(c.HostPath)
}

func validateHostPath(hostpath *central.HostPathPersistence) error {
	if hostpath == nil {
		return nil
	}
	if (hostpath.NodeSelectorKey == "") != (hostpath.NodeSelectorValue == "") {
		return fmt.Errorf("Both node selector key and node selector value must be specified when using a hostpath")
	}
	return nil
}

func validateExternal(ext *central.ExternalPersistence, cluster v1.ClusterType) error {
	if ext == nil {
		return nil
	}
	if cluster == v1.ClusterType_SWARM_CLUSTER && ext.Name == "" {
		return fmt.Errorf("name must be specified for external volume in Swarm")
	}
	return nil
}
