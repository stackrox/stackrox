package resolver

import "github.com/stackrox/rox/sensor/common/store"

// DeploymentReference generates a list of deployment IDs that need to be updated given the deployment store.
type DeploymentReference func(store store.DeploymentStore) []string

// ResolveDeploymentIds is an identify function that simply returns a list of deployment ids passed
func ResolveDeploymentIds(ids ...string) DeploymentReference {
	return func(_ store.DeploymentStore) []string {
		return ids
	}
}
