package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetServiceAccount returns a mock Service Account
func GetServiceAccount() *storage.ServiceAccount {
	return &storage.ServiceAccount{
		Id:          uuid.NewDummy().String(),
		ClusterId:   fixtureconsts.Cluster1,
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
