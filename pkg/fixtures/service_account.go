package fixtures

import (
	"github.com/stackrox/stackrox/generated/storage"
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
