package resolver

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/store"
)

// DeploymentReference generates a list of deployment IDs that need to be updated given the deployment store.
type DeploymentReference func(store store.DeploymentStore) []string

// ResolveDeploymentIds is an identify function that simply returns a list of deployment ids passed
func ResolveDeploymentIds(ids ...string) DeploymentReference {
	return func(_ store.DeploymentStore) []string {
		return ids
	}
}

// NamespaceServiceAccount is a helper struct that represents an object that has both a Namespace and a Service Account.
type NamespaceServiceAccount struct {
	Namespace, ServiceAccount string
}

// ResolveDeploymentsByMultipleServiceAccounts returns a list of deployment IDs given a list of ServiceAccounts
func ResolveDeploymentsByMultipleServiceAccounts(serviceAccounts []NamespaceServiceAccount) DeploymentReference {
	return func(store store.DeploymentStore) []string {
		var allIds []string
		for _, sa := range serviceAccounts {
			allIds = append(allIds, store.FindDeploymentIDsWithServiceAccount(sa.Namespace, sa.ServiceAccount)...)
		}
		return allIds
	}
}

// ResolveDeploymentLabels returns a function that returns a list of deployment ids based on namespace and labels
func ResolveDeploymentLabels(namespace string, sel selector.Selector) DeploymentReference {
	return func(store store.DeploymentStore) []string {
		return store.FindDeploymentIDsByLabels(namespace, sel)
	}
}

// ResolveAllDeployments returns a function that generates a list of a all deployment ids in the system
func ResolveAllDeployments() DeploymentReference {
	return func(store store.DeploymentStore) []string {
		allDeployments := store.GetAll()
		ids := make([]string, len(allDeployments))
		for i, dp := range allDeployments {
			ids[i] = dp.GetId()
		}
		return ids
	}
}

// ResolveDeploymentsByImages returns a function that returns a list of deployment ids based on a slice of images
func ResolveDeploymentsByImages(images ...*storage.Image) DeploymentReference {
	return func(store store.DeploymentStore) []string {
		return store.FindDeploymentIDsByImages(images)
	}
}
