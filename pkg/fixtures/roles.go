package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetScopedK8SRole returns a mock K8SRole belonging to the input scope.
func GetScopedK8SRole(id string, clusterID string, namespace string) *storage.K8SRole {
	return &storage.K8SRole{
		Id:        id,
		Namespace: namespace,
		ClusterId: clusterID,
	}
}

// GetSACTestK8SRoleSet returns a set of mock K8SRoleBindings that can be used
// for scoped access control sets.
// It will include:
// 9 Process indicators scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster3, 3 to each Namespace A / B / C.
func GetSACTestK8SRoleSet() []*storage.K8SRole {
	return []*storage.K8SRole{
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster1, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster2, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceA),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceB),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceC),
		scopedK8SRole(testconsts.Cluster3, testconsts.NamespaceC),
	}
}

func scopedK8SRole(clusterID, namespace string) *storage.K8SRole {
	return GetScopedK8SRole(uuid.NewV4().String(), clusterID, namespace)
}
