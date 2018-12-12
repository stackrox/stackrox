package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetSecret returns a mock Secret
func GetSecret() *storage.Secret {
	return &storage.Secret{
		Id:          "ID",
		ClusterId:   "clusterid",
		ClusterName: "clustername",
	}
}
