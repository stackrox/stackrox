package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetNetworkBaseline returns a mock network baseline.
func GetNetworkBaseline() *storage.NetworkBaseline {
	return GetScopedNetworkBaseline(GetDeployment().GetId(), fixtureconsts.Cluster1, "stackrox")
}

// GetScopedNetworkBaseline returns a mock network baseline belonging to the input scope.
func GetScopedNetworkBaseline(id, clusterID, namespace string) *storage.NetworkBaseline {
	return &storage.NetworkBaseline{
		DeploymentId:         id,
		ClusterId:            clusterID,
		Namespace:            namespace,
		Peers:                nil,
		ForbiddenPeers:       nil,
		ObservationPeriodEnd: nil,
		Locked:               false,
		DeploymentName:       GetDeployment().GetName(),
	}
}

// GetSACTestNetworkBaseline returns a set of mock network baselines that can be
// used for scoped access control tests.
func GetSACTestNetworkBaseline() []*storage.NetworkBaseline {
	return []*storage.NetworkBaseline{
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceA),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster1, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster2, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster2, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster2, testconsts.NamespaceB),
		scopedNetworkBaseline(testconsts.Cluster2, testconsts.NamespaceC),
		scopedNetworkBaseline(testconsts.Cluster2, testconsts.NamespaceC),
	}
}

func scopedNetworkBaseline(clusterID, namespace string) *storage.NetworkBaseline {
	return GetScopedNetworkBaseline(uuid.NewV4().String(), clusterID, namespace)
}
