package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetScopedK8SRoleBinding returns a mock K8SRoleBinding belonging to the input scope.
func GetScopedK8SRoleBinding(id string, clusterID string, namespace string) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:          id,
		Namespace:   namespace,
		ClusterId:   clusterID,
		ClusterRole: false,
		Subjects: []*storage.Subject{{
			Id:        id,
			Namespace: namespace,
			ClusterId: clusterID,
		}},
		RoleId: id,
	}
}

// GetSACTestK8SRoleBindingSet returns a set of mock K8SRoleBindings that can be used
// for scoped access control sets.
// It will include:
// 9 Process indicators scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster3, 3 to each Namespace A / B / C.
func GetSACTestK8SRoleBindingSet() []*storage.K8SRoleBinding {
	return []*storage.K8SRoleBinding{
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster1, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster2, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceA),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceB),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceC),
		scopedK8SRoleBinding(testconsts.Cluster3, testconsts.NamespaceC),
	}
}

func scopedK8SRoleBinding(clusterID, namespace string) *storage.K8SRoleBinding {
	return GetScopedK8SRoleBinding(uuid.NewV4().String(), clusterID, namespace)
}
