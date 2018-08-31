package fixtures

import "github.com/stackrox/rox/generated/api/v1"

// GetSecret returns a mock Secret
func GetSecret() *v1.Secret {
	return &v1.Secret{
		ClusterId:   "clusterid",
		ClusterName: "clustername",
	}
}
