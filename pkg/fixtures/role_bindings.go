package fixtures

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// GetScopedK8SRoleBinding returns a mock K8SRoleBinding belonging to the input scope.
func GetScopedK8SRoleBinding(id string, clusterID string, namespace string) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:        id,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}
