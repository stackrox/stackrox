package fixtures

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// GetScopedK8SRole returns a mock K8SRole belonging to the input scope.
func GetScopedK8SRole(id string, clusterID string, namespace string) *storage.K8SRole {
	return &storage.K8SRole{
		Id:        id,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}
