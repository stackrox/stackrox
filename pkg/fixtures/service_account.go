package fixtures

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
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
func GetScopedServiceAccount(t *testing.T, id string, clusterID string, namespace string) *storage.ServiceAccount {
	svcAccount := &storage.ServiceAccount{}
	require.NoError(t, testutils.FullInit(svcAccount, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	svcAccount.ClusterId = clusterID
	svcAccount.Namespace = namespace
	svcAccount.Id = id
	return svcAccount
}
