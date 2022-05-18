package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetServiceAccount returns a mock Service Account
func GetServiceAccount() *storage.ServiceAccount {
	return &storage.ServiceAccount{
		Id:          "ID",
		ClusterId:   "clusterid",
		ClusterName: "clustername",
		Namespace:   "namespace",
	}
}

// GetScopedServiceAccount returns a mock ServiceAccount belonging to the input scope.
func GetScopedServiceAccount(id string, clusterID string, namespace string) *storage.ServiceAccount {
	return &storage.ServiceAccount{
		Id:        id,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}

// GetSACTestServiceAccountSet returns a set of mock ServiceAccounts that can be used
// for scoped access control sets.
// It will include:
// 9 ServiceAccounts scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 ServiceAccounts scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 ServiceAccounts scoped to Cluster3, 3 to each Namespace A / B / C.
func GetSACTestServiceAccountSet() []*storage.ServiceAccount {
	return []*storage.ServiceAccount{
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster1, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster2, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceA),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceB),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceC),
		scopedServiceAccount(testconsts.Cluster3, testconsts.NamespaceC),
	}
}

func scopedServiceAccount(clusterID, namespace string) *storage.ServiceAccount {
	return GetScopedServiceAccount(uuid.NewV4().String(), clusterID, namespace)
}
