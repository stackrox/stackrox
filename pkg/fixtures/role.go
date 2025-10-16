package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetScopedK8SRole returns a mock K8SRole belonging to the input scope.
func GetScopedK8SRole(id string, clusterID string, namespace string) *storage.K8SRole {
	k8SRole := &storage.K8SRole{}
	k8SRole.SetId(id)
	k8SRole.SetClusterId(clusterID)
	k8SRole.SetNamespace(namespace)
	return k8SRole
}

// GetMultipleK8SRoles returns a given number of roles.
// The cluster role property will toggle from false to true.
func GetMultipleK8SRoles(numRoles int) []*storage.K8SRole {
	roles := make([]*storage.K8SRole, 0, numRoles)
	clusterRole := false
	for i := 0; i < numRoles; i++ {
		k8SRole := &storage.K8SRole{}
		k8SRole.SetId(uuid.NewV4().String())
		k8SRole.SetName(fmt.Sprintf("role%d", i))
		k8SRole.SetNamespace(fmt.Sprintf("namespace%d", i))
		k8SRole.SetClusterId(uuid.NewV4().String())
		k8SRole.SetClusterName(fmt.Sprintf("cluster%d", i))
		k8SRole.SetClusterRole(clusterRole)
		roles = append(roles, k8SRole)
		clusterRole = !clusterRole
	}
	return roles
}
