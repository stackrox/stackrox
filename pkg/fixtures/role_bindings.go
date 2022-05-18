package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetScopedK8SRoleBinding returns a mock K8SRoleBinding belonging to the input scope.
func GetScopedK8SRoleBinding(id string, clusterID string, namespace string) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:          id,
		Namespace:   namespace,
		ClusterId:   clusterID,
		ClusterRole: false,
		Subjects: []*storage.Subject{{
			Id:        id,
			Namespace: namespace,
			ClusterId: clusterID,
		}},
		RoleId: id,
	}
}
