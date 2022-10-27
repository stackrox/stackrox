package resources

import "github.com/stackrox/rox/pkg/sync"

var (
	dsInit   sync.Once
	depStore *DeploymentStore

	saInit  sync.Once
	saStore *ServiceAccountStore

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

// ServiceAccountStoreSingleton returns a singleton of the ServiceAccountStore
func ServiceAccountStoreSingleton() *ServiceAccountStore {
	saInit.Do(func() {
		saStore = newServiceAccountStore()
	})
	return saStore
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
