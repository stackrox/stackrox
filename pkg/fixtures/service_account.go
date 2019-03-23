package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
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
