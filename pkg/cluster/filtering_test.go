package cluster

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var cluster = &storage.Cluster{
	Name:               "cluster-name",
	MainImage:          "stackrox.io/main:3.0.55.0",
	CentralApiEndpoint: "central.stackrox:443",
	Type:               storage.ClusterType_OPENSHIFT4_CLUSTER,
	ManagedBy:          storage.ManagerType_MANAGER_TYPE_HELM_CHART,
	HelmConfig: &storage.CompleteClusterConfig{
		DynamicConfig: &storage.DynamicClusterConfig{
			RuntimeDataControl: &storage.DynamicClusterConfig_RuntimeDataControl{
				Persistence: true,
			},
		},
	},
}

func TestNamespaceFilter(t *testing.T) {
	cases := map[string]struct {
		configureClusterFn func(*storage.Cluster)
		expectedFilter     string
	}{
		"Empty filter configuration": {
			configureClusterFn: func(*storage.Cluster) {},
			expectedFilter:     "",
		},
		"Custom filter configuration": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.NamespaceFilter = "test-.*"
			},
			expectedFilter: "test-.*",
		},
		"No openshift": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.ExcludeOpenshift = true
			},
			expectedFilter: "openshift-.*",
		},
		"Custom filter and no openshift": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.NamespaceFilter = "test-.*"
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.ExcludeOpenshift = true
			},
			expectedFilter: "test-.*|openshift-.*",
		},
		"Custom filter and no persistence": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.NamespaceFilter = "test-.*"
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.Persistence = false
			},
			expectedFilter: ".*",
		},
		"No persistence and no openshift": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.ExcludeOpenshift = true
				cluster.HelmConfig.DynamicConfig.RuntimeDataControl.Persistence = false
			},
			expectedFilter: ".*",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cluster := cluster.CloneVT()
			c.configureClusterFn(cluster)

			filter := GetNamespaceFilter(cluster)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}
