package centralclient

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
)

func Test_clusterRegistered(t *testing.T) {
	// clean the global set of uninitialized clusters:
	uninitializedClusters = set.NewSet[string]()

	props := map[string]any{}

	rp := &phonehome.RequestParams{
		Path: "random",
	}
	assert.False(t, clusterRegistered(rp, props))
	assert.Empty(t, props)

	rp = &phonehome.RequestParams{
		Path: "/v1.ClustersService/PostCluster",
	}
	assert.True(t, clusterRegistered(rp, props))
	assert.Equal(t, map[string]any{"Code": 0}, props)

	rp.GRPCReq = &storage.Cluster{
		Type:      storage.ClusterType_GENERIC_CLUSTER,
		Id:        "cluster-id",
		ManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL,
		HealthStatus: &storage.ClusterHealthStatus{
			SensorHealthStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
	}
	assert.False(t, uninitializedClusters.Contains("cluster-id"))
	// remembers the uninitialized cluster in memory:
	assert.True(t, clusterRegistered(rp, props))
	assert.Equal(t, map[string]any{
		"Code":         0,
		"Cluster ID":   "cluster-id",
		"Cluster Type": "GENERIC_CLUSTER",
		"Managed By":   "MANAGER_TYPE_MANUAL",
	}, props)
	assert.True(t, uninitializedClusters.Contains("cluster-id"))
}

func Test_clusterInitialized(t *testing.T) {
	// clean the global set of uninitialized clusters:
	uninitializedClusters = set.NewSet[string]()

	props := map[string]any{}

	rp := &phonehome.RequestParams{
		Path: "random",
	}
	assert.False(t, clusterInitialized(rp, props))
	assert.Empty(t, props)
	assert.False(t, uninitializedClusters.Contains("cluster-id"))

	rp = &phonehome.RequestParams{
		GRPCReq: &storage.Cluster{
			Type:      storage.ClusterType_GENERIC_CLUSTER,
			Id:        "cluster-id",
			ManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL,
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNINITIALIZED,
			},
		}}

	rp.Path = "/v1.ClustersService/PostCluster"
	// remembers the uninitialized cluster in memory:
	assert.True(t, clusterRegistered(rp, props))
	assert.True(t, uninitializedClusters.Contains("cluster-id"))

	rp.GRPCReq.(*storage.Cluster).HealthStatus.SensorHealthStatus = storage.ClusterHealthStatus_HEALTHY

	rp.Path = "/v1.ClustersService/PutCluster"
	// removes the now initialized cluster from memory:
	assert.True(t, clusterInitialized(rp, props), "Should fire because the sensor is healthy")
	assert.Equal(t, map[string]any{
		"Code":         0,
		"Cluster ID":   "cluster-id",
		"Cluster Type": "GENERIC_CLUSTER",
		"Managed By":   "MANAGER_TYPE_MANUAL",
	}, props)
	assert.False(t, uninitializedClusters.Contains("cluster-id"))

	props = map[string]any{}
	assert.False(t, clusterInitialized(rp, props), "Should not fire, as the cluster is forgotten already")
	assert.Empty(t, props)

	rp.GRPCReq.(*storage.Cluster).HealthStatus.SensorHealthStatus = storage.ClusterHealthStatus_UNINITIALIZED
	// adds cluster back to memory:
	assert.False(t, clusterInitialized(rp, props), "Should not fire because the known sensor is UNINITIALIZED again")
	assert.True(t, uninitializedClusters.Contains("cluster-id"))

	rp.GRPCReq.(*storage.Cluster).HealthStatus.SensorHealthStatus = storage.ClusterHealthStatus_DEGRADED
	assert.True(t, clusterInitialized(rp, props), "Should fire again because the sensor is somehow initialized")
}
