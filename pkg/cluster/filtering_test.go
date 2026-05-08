package cluster

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
)

var cluster = &storage.Cluster{
	Name:               "cluster-name",
	MainImage:          "stackrox.io/main:3.0.55.0",
	CentralApiEndpoint: "central.stackrox:443",
	Type:               storage.ClusterType_OPENSHIFT4_CLUSTER,
	DynamicConfig: &storage.DynamicClusterConfig{
		ProcessIndicators: &storage.DynamicClusterConfig_ProcessIndicatorsConfig{
			NoPersistence: false,
		},
	},
	HelmConfig: &storage.CompleteClusterConfig{
		DynamicConfig: &storage.DynamicClusterConfig{
			ProcessIndicators: &storage.DynamicClusterConfig_ProcessIndicatorsConfig{
				NoPersistence: false,
			},
		},
	},
}

func TestNamespaceFilter(t *testing.T) {
	cases := map[string]struct {
		configureClusterFn func(*storage.Cluster)
		expectedFilter     *string
	}{
		"Empty filter configuration": {
			configureClusterFn: func(*storage.Cluster) {},
			expectedFilter:     nil,
		},
		"Custom filter configuration": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "test-.*"

				cluster.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "test-.*"
			},
			expectedFilter: pointers.String("test-.*"),
		},
		"No openshift": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeOpenshiftNs = true

				cluster.DynamicConfig.ProcessIndicators.ExcludeOpenshiftNs = true
			},
			expectedFilter: pointers.String("^openshift$|^openshift-.*"),
		},
		"Custom filter and no openshift": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "test-.*"
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeOpenshiftNs = true

				cluster.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "test-.*"
				cluster.DynamicConfig.ProcessIndicators.ExcludeOpenshiftNs = true
			},
			expectedFilter: pointers.String("test-.*|^openshift$|^openshift-.*"),
		},
		"Custom filter and no persistence": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "test-.*"
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.NoPersistence = true

				cluster.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "test-.*"
				cluster.DynamicConfig.ProcessIndicators.NoPersistence = true
			},
			expectedFilter: pointers.String(".*"),
		},
		"No persistence and no openshift": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeOpenshiftNs = true
				cluster.HelmConfig.DynamicConfig.ProcessIndicators.NoPersistence = true

				cluster.DynamicConfig.ProcessIndicators.ExcludeOpenshiftNs = true
				cluster.DynamicConfig.ProcessIndicators.NoPersistence = true
			},
			expectedFilter: pointers.String(".*"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			// Helm managed cluster
			cluster.ManagedBy = storage.ManagerType_MANAGER_TYPE_HELM_CHART
			cluster := cluster.CloneVT()
			c.configureClusterFn(cluster)

			filter := GetNamespaceFilter(cluster)
			assert.Equal(t, c.expectedFilter, filter)

			// Manually managed cluster
			cluster.ManagedBy = storage.ManagerType_MANAGER_TYPE_MANUAL
			cluster = cluster.CloneVT()
			c.configureClusterFn(cluster)

			filter = GetNamespaceFilter(cluster)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}
