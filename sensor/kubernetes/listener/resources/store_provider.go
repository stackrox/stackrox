package resources

import (
	"github.com/stackrox/rox/pkg/registrymirror"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
)

// StoreProvider holds all stores used in sensor and exposes a public interface for each that can be used outside of the listeners.
type StoreProvider struct {
	deploymentStore        *DeploymentStore
	podStore               *PodStore
	serviceStore           *serviceStore
	networkPolicyStore     *networkPolicyStoreImpl
	rbacStore              rbac.Store
	serviceAccountStore    *ServiceAccountStore
	endpointManager        endpointManager
	nodeStore              *nodeStoreImpl
	entityStore            *clusterentities.Store
	orchestratorNamespaces *orchestratornamespaces.OrchestratorNamespaces
	registryStore          *registry.Store
	registryMirrorStore    registrymirror.Store
	nsStore                *namespaceStore

	cleanableStores []CleanableStore
}

// CleanableStore defines a store implementation that has a function for deleting all entries
type CleanableStore interface {
	Cleanup()
}

// InitializeStore creates the store instances
func InitializeStore(hm clusterentities.HeritageManager) *StoreProvider {
	memSizeSetting := pastClusterEntitiesMemorySize.IntegerSetting()
	if memSizeSetting < 0 {
		memSizeSetting = pastClusterEntitiesMemorySize.DefaultValue()
	}
	log.Infof("Initializing cluster entities store with memory that will last for %d ticks", memSizeSetting)
	deployStore := newDeploymentStore()
	podStore := newPodStore()
	svcStore := newServiceStore()
	nodeStore := newNodeStore()
	entityStore := clusterentities.NewStore(uint16(memSizeSetting), hm, debugClusterEntitiesStore.BooleanSetting())
	if debugClusterEntitiesStore.BooleanSetting() {
		go entityStore.StartDebugServer()
	}

	endpointManager := newEndpointManager(svcStore, deployStore, podStore, nodeStore, entityStore)
	p := &StoreProvider{
		deploymentStore:        deployStore,
		podStore:               podStore,
		serviceStore:           svcStore,
		nodeStore:              nodeStore,
		entityStore:            entityStore,
		endpointManager:        endpointManager,
		networkPolicyStore:     newNetworkPoliciesStore(),
		rbacStore:              rbac.NewStore(),
		serviceAccountStore:    newServiceAccountStore(),
		orchestratorNamespaces: orchestratornamespaces.NewOrchestratorNamespaces(),
		registryStore:          registry.NewRegistryStore(nil),
		registryMirrorStore:    registrymirror.NewFileStore(),
		nsStore:                newNamespaceStore(),
	}

	p.cleanableStores = []CleanableStore{
		p.deploymentStore,
		p.podStore,
		p.serviceStore,
		p.nodeStore,
		p.entityStore,
		p.networkPolicyStore,
		p.rbacStore,
		p.serviceAccountStore,
		p.orchestratorNamespaces,
		p.registryStore,
		p.registryMirrorStore,
		p.nsStore,
	}

	return p
}

// CleanupStores deletes all entries from all stores
func (p *StoreProvider) CleanupStores() {
	for _, cleanable := range p.cleanableStores {
		cleanable.Cleanup()
	}
}

// Deployments returns the deployment store public interface
func (p *StoreProvider) Deployments() store.DeploymentStore {
	return p.deploymentStore
}

// Pods returns the pod store public interface
func (p *StoreProvider) Pods() store.PodStore {
	return p.podStore
}

// Services returns the service store public interface
func (p *StoreProvider) Services() store.ServiceStore {
	return p.serviceStore
}

// NetworkPolicies returns the network policy store public interface
func (p *StoreProvider) NetworkPolicies() store.NetworkPolicyStore {
	return p.networkPolicyStore
}

// RBAC returns the RBAC store public interface
func (p *StoreProvider) RBAC() store.RBACStore {
	return p.rbacStore
}

// ServiceAccounts returns the ServiceAccount store public interface
func (p *StoreProvider) ServiceAccounts() store.ServiceAccountStore {
	return p.serviceAccountStore
}

// EndpointManager returns the EndpointManager public interface
func (p *StoreProvider) EndpointManager() store.EndpointManager {
	return p.endpointManager
}

// Registries returns the Registry store public interface
func (p *StoreProvider) Registries() *registry.Store {
	return p.registryStore
}

// Entities returns the cluster entities store public interface
func (p *StoreProvider) Entities() *clusterentities.Store {
	return p.entityStore
}

// Nodes returns the Nodes public interface
func (p *StoreProvider) Nodes() store.NodeStore {
	return p.nodeStore
}

// RegistryMirrors returns the RegistryMirror store public interface.
func (p *StoreProvider) RegistryMirrors() registrymirror.Store {
	return p.registryMirrorStore
}
