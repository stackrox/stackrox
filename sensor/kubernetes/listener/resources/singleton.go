package resources

import "github.com/stackrox/stackrox/pkg/sync"

var (
	dsInit   sync.Once
	depStore *DeploymentStore

	psInit   sync.Once
	podStore *PodStore

	netpolInit  sync.Once
	netpolStore *networkPolicyStoreImpl
)

// DeploymentStoreSingleton returns a singleton of the DeploymentStore
func DeploymentStoreSingleton() *DeploymentStore {
	dsInit.Do(func() {
		depStore = newDeploymentStore()
	})
	return depStore
}

// PodStoreSingleton returns a singleton of the PodStore
func PodStoreSingleton() *PodStore {
	psInit.Do(func() {
		podStore = newPodStore()
	})
	return podStore
}

// NetworkPolicySingleton returns a singleton of NetworkPolicyStore
func NetworkPolicySingleton() *networkPolicyStoreImpl {
	netpolInit.Do(func() {
		netpolStore = newNetworkPoliciesStore()
	})
	return netpolStore
}
