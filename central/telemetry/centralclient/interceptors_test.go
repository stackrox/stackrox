package centralclient

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/suite"
)

func TestInterceptors(t *testing.T) {
	suite.Run(t, new(interceptorsTest))
}

var _ suite.SetupTestSuite = (*interceptorsTest)(nil)

type interceptorsTest struct {
	suite.Suite
}

func (t *interceptorsTest) SetupTest() {
	// clean the global set of uninitialized clusters:
	uninitializedClusters = set.NewSet[string]()
}

func (t *interceptorsTest) TestClusterRegisteredNoFire() {
	props := map[string]any{}

	// Path is not "/v1.ClustersService/PostCluster":
	rp := &phonehome.RequestParams{
		Path: "random",
	}
	t.False(clusterRegistered(rp, props), "must not fire, as Path doesn't match")
	t.Empty(props, "props must not be touched")
}

func (t *interceptorsTest) TestClusterRegisteredFire() {
	props := map[string]any{}

	// Test for matching Path:
	rp := &phonehome.RequestParams{
		Path: "/v1.ClustersService/PostCluster",
	}
	t.True(clusterRegistered(rp, props), "must fire, as Path matches")
	t.Equal(map[string]any{"Code": 0}, props, "props must have only Code, as gRPC request details are not provided")

	// Test with gRPC request details:
	rp.GRPCReq = &storage.Cluster{
		Type:      storage.ClusterType_GENERIC_CLUSTER,
		Id:        "cluster-id",
		ManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL,
		HealthStatus: &storage.ClusterHealthStatus{
			SensorHealthStatus: storage.ClusterHealthStatus_UNINITIALIZED,
		},
	}
	t.False(uninitializedClusters.Contains("cluster-id"), "cluster-id must not be registered as uninitialized yet")
	// remembers the uninitialized cluster in memory:
	t.True(clusterRegistered(rp, props), "must fire, as Path matches, and cluster-id has not been registered")
	t.Equal(map[string]any{
		"Code":         0,
		"Cluster ID":   "cluster-id",
		"Cluster Type": "GENERIC_CLUSTER",
		"Managed By":   "MANAGER_TYPE_MANUAL",
	}, props, "props must have all the fields from the gRPC request details")
	t.True(uninitializedClusters.Contains("cluster-id"), "cluster-id must be registered as uninitialized now")
}

func (t *interceptorsTest) TestClusterInitializedNoFire() {
	props := map[string]any{}

	// Path is not "/v1.ClustersService/PutCluster":
	rp := &phonehome.RequestParams{
		Path: "random",
	}
	t.False(clusterInitialized(rp, props), "must not fire, as Path doesn't match")
	t.Empty(props)
	t.False(uninitializedClusters.Contains("cluster-id"))
}

func (t *interceptorsTest) TestClusterInitializedFire() {
	// Register the uninitialized cluster:
	uninitializedClusters.Add("cluster-id")

	props := map[string]any{}
	rp := &phonehome.RequestParams{
		Path: "/v1.ClustersService/PutCluster",
		GRPCReq: &storage.Cluster{
			Type:      storage.ClusterType_GENERIC_CLUSTER,
			Id:        "cluster-id",
			ManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL,
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			},
		}}

	// removes the now initialized cluster from memory:
	t.True(clusterInitialized(rp, props), "must fire because the sensor is healthy")
	t.False(uninitializedClusters.Contains("cluster-id"))

	t.Equal(map[string]any{
		"Code":         0,
		"Cluster ID":   "cluster-id",
		"Cluster Type": "GENERIC_CLUSTER",
		"Managed By":   "MANAGER_TYPE_MANUAL",
	}, props)

	props = map[string]any{}
	t.False(clusterInitialized(rp, props), "must not fire, as the cluster is forgotten already")
	t.Empty(props)

	rp.GRPCReq.(*storage.Cluster).HealthStatus.SensorHealthStatus = storage.ClusterHealthStatus_UNINITIALIZED
	// adds cluster back to memory:
	t.False(clusterInitialized(rp, props), "must not fire because the known sensor is UNINITIALIZED again")
	t.True(uninitializedClusters.Contains("cluster-id"))

	rp.GRPCReq.(*storage.Cluster).HealthStatus.SensorHealthStatus = storage.ClusterHealthStatus_DEGRADED
	t.True(clusterInitialized(rp, props), "must fire again because the sensor is somehow initialized")
}
