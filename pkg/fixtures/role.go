package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetScopedK8SRole returns a mock K8SRole belonging to the input scope.
func GetScopedK8SRole(id string, clusterID string, namespace string) *storage.K8SRole {
	return &storage.K8SRole{
		Id:        id,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}

// GetMultipleK8SRoles returns a given number of roles.
// The cluster role property will toggle from false to true.
func GetMultipleK8SRoles(numRoles int) []*storage.K8SRole {
	roles := make([]*storage.K8SRole, 0, numRoles)
	clusterRole := false
	for i := 0; i < numRoles; i++ {
		roles = append(roles, &storage.K8SRole{
			Id:          uuid.NewV4().String(),
			Name:        fmt.Sprintf("role%d", i),
			Namespace:   fmt.Sprintf("namespace%d", i),
			ClusterId:   uuid.NewV4().String(),
			ClusterName: fmt.Sprintf("cluster%d", i),
			ClusterRole: clusterRole,
		})
		clusterRole = !clusterRole
	}
	return roles
}
