package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetServiceAccount returns a mock Service Account
func GetServiceAccount() *storage.ServiceAccount {
	sa := &storage.ServiceAccount{}
	sa.SetId(uuid.NewDummy().String())
	sa.SetClusterId(fixtureconsts.Cluster1)
	sa.SetClusterName("clustername")
	sa.SetNamespace("namespace")
	return sa
}

// GetScopedServiceAccount returns a mock ServiceAccount belonging to the input scope.
func GetScopedServiceAccount(id string, clusterID string, namespace string) *storage.ServiceAccount {
	sa := &storage.ServiceAccount{}
	sa.SetId(id)
	sa.SetClusterId(clusterID)
	sa.SetNamespace(namespace)
	return sa
}
