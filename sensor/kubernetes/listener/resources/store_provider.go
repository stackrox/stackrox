package resources

import (
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
)

// InMemoryStoreProvider holds all stores used in sensor and exposes a public interface for each that can be used outside of the listeners.
type InMemoryStoreProvider struct {
	deploymentStore *DeploymentStore
	podStore        *PodStore
	serviceStore    *serviceStore
	rbacStore       rbac.Store
	endpointManager endpointManager
	nodeStore       *nodeStore
	entityStore     *clusterentities.Store
}

// InitializeStore creates the store instances
func InitializeStore() *InMemoryStoreProvider {
	deployStore := DeploymentStoreSingleton()
	podStore := PodStoreSingleton()
	svcStore := newServiceStore()
	rbacStore := rbac.NewStore()
	nodeStore := newNodeStore()
	entityStore := clusterentities.StoreInstance()
	endpointManager := newEndpointManager(svcStore, deployStore, podStore, nodeStore, entityStore)
	return &InMemoryStoreProvider{
		deploymentStore: deployStore,
		podStore:        podStore,
		serviceStore:    svcStore,
		rbacStore:       rbacStore,
		nodeStore:       nodeStore,
		entityStore:     entityStore,
		endpointManager: endpointManager,
	}
}

// Deployments returns the deployment store public interface
func (p *InMemoryStoreProvider) Deployments() store.DeploymentStore {
	return p.deploymentStore
}

// Services returns the service store public interface
func (p *InMemoryStoreProvider) Services() store.ServiceStore {
	return p.serviceStore
}

// RBAC returns the RBAC store public interface
func (p *InMemoryStoreProvider) RBAC() store.RBACStore {
	return p.rbacStore
}

// EndpointManager returns the EndpointManager public interface
func (p *InMemoryStoreProvider) EndpointManager() store.EndpointManager {
	return p.endpointManager
}
