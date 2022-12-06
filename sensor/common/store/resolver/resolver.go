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

// ResolveDeploymentsByServiceAccount returns the deployments matching a certain service account
func ResolveDeploymentsByServiceAccount(namespace, serviceAccount string) DeploymentReference {
	return func(store store.DeploymentStore) []string {
		return store.FindDeploymentsWithServiceAccount(namespace, serviceAccount)
	}
}

type NamespaceServiceAccount struct {
	Namespace, ServiceAccount string
}

// ResolveDeploymentsByMultipleServiceAccounts a
func ResolveDeploymentsByMultipleServiceAccounts(serviceAccounts []NamespaceServiceAccount) DeploymentReference {
	return func(store store.DeploymentStore) []string {
		var allIds []string
		for _, sa := range serviceAccounts {
			allIds = append(allIds, store.FindDeploymentsWithServiceAccount(sa.Namespace, sa.ServiceAccount)...)
		}
		return allIds
	}
}
