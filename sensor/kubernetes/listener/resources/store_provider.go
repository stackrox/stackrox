package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/reconcile"
	"github.com/stackrox/rox/pkg/registrymirror"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/reconciliation/types"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
)

var (
	errUnableToReconcile                        = errors.New("unable to reconcile resource")
	_                    reconcile.Reconcilable = (*InMemoryStoreProvider)(nil)
	_                    reconcile.Resource     = (*reconcilePair)(nil)
)

type reconcilePair struct {
	resID   string
	resType string
}

func (r *reconcilePair) GetPair() (string, string) {
	return r.resID, r.resType
}

type reconcileDeleteFn func(resType, resID string, resHash uint64) ([]reconcile.Resource, error)

// InMemoryStoreProvider holds all stores used in sensor and exposes a public interface for each that can be used outside of the listeners.
type InMemoryStoreProvider struct {
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

	cleanableStores    []CleanableStore
	reconcilePodStore  types.PodStore
	reconcilableStores map[string]reconcileDeleteFn
}

var _ store.Provider = (*InMemoryStoreProvider)(nil)

// CleanableStore defines a store implementation that has a function for deleting all entries
type CleanableStore interface {
	Cleanup()
}

// InitializeStore creates the store instances
func InitializeStore() *InMemoryStoreProvider {
	deployStore := newDeploymentStore()
	podStore := newPodStore()
	svcStore := newServiceStore()
	nodeStore := newNodeStore()
	entityStore := clusterentities.NewStore()
	endpointManager := newEndpointManager(svcStore, deployStore, podStore, nodeStore, entityStore)
	reconciliationPodStore := types.NewPodStore()
	p := &InMemoryStoreProvider{
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
		reconcilePodStore:      reconciliationPodStore,
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
	}
	p.reconcilableStores = map[string]reconcileDeleteFn{
		deduper.TypeDeployment.String():     p.deploymentStore.ReconcileDelete,
		deduper.TypePod.String():            reconcileDeleteForPods(p.reconcilePodStore, p.podStore, p.deploymentStore),
		deduper.TypeServiceAccount.String(): p.serviceAccountStore.ReconcileDelete,
		deduper.TypeSecret.String():         p.registryStore.ReconcileDelete,
		deduper.TypeNode.String():           p.nodeStore.ReconcileDelete,
		deduper.TypeNetworkPolicy.String():  p.networkPolicyStore.ReconcileDelete,
		deduper.TypeRole.String():           p.rbacStore.ReconcileDelete,
		deduper.TypeBinding.String():        p.rbacStore.ReconcileDelete,
	}

	return p
}

// CleanupStores deletes all entries from all stores
func (p *InMemoryStoreProvider) CleanupStores() {
	for _, cleanable := range p.cleanableStores {
		cleanable.Cleanup()
	}
}

// Deployments returns the deployment store public interface
func (p *InMemoryStoreProvider) Deployments() store.DeploymentStore {
	return p.deploymentStore
}

// Pods returns the pod store public interface
func (p *InMemoryStoreProvider) Pods() store.PodStore {
	return p.podStore
}

// Services returns the service store public interface
func (p *InMemoryStoreProvider) Services() store.ServiceStore {
	return p.serviceStore
}

// NetworkPolicies returns the network policy store public interface
func (p *InMemoryStoreProvider) NetworkPolicies() store.NetworkPolicyStore {
	return p.networkPolicyStore
}

// RBAC returns the RBAC store public interface
func (p *InMemoryStoreProvider) RBAC() store.RBACStore {
	return p.rbacStore
}

// ServiceAccounts returns the ServiceAccount store public interface
func (p *InMemoryStoreProvider) ServiceAccounts() store.ServiceAccountStore {
	return p.serviceAccountStore
}

// EndpointManager returns the EndpointManager public interface
func (p *InMemoryStoreProvider) EndpointManager() store.EndpointManager {
	return p.endpointManager
}

// Registries returns the Registry store public interface
func (p *InMemoryStoreProvider) Registries() *registry.Store {
	return p.registryStore
}

// Entities returns the cluster entities store public interface
func (p *InMemoryStoreProvider) Entities() *clusterentities.Store {
	return p.entityStore
}

// Nodes returns the Nodes public interface
func (p *InMemoryStoreProvider) Nodes() store.NodeStore {
	return p.nodeStore
}

// RegistryMirrors returns the RegistryMirror store public interface.
func (p *InMemoryStoreProvider) RegistryMirrors() registrymirror.Store {
	return p.registryMirrorStore
}

func (p *InMemoryStoreProvider) ReconciliationPodStore() types.PodStore {
	return p.reconcilePodStore
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliation ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (p *InMemoryStoreProvider) ReconcileDelete(resType, resID string, resHash uint64) ([]reconcile.Resource, error) {
	if reconcileFn, found := p.reconcilableStores[resType]; found {
		return reconcileFn(resType, resID, resHash)
	}
	return nil, errors.Wrapf(errUnableToReconcile, "Don't know how to reconcile resource type %q", resType)
}

// reconcileDeleteForPods tries to reconcile deleted pods in the pod store and the deployment store.
// This is needed due to rogue pods being treated as deployment objects.
func reconcileDeleteForPods(reconciliationStore types.PodStore, podStore reconcile.Reconcilable, depStore reconcile.Reconcilable) func(resType, resID string, resHash uint64) ([]reconcile.Resource, error) {
	return func(resType, resID string, resHash uint64) ([]reconcile.Resource, error) {
		errList := errorhelpers.NewErrorList("PodReconcileDelete")
		var ret []reconcile.Resource
		idFromPodStore, err := podStore.ReconcileDelete(resType, resID, resHash)
		if err != nil {
			errList.AddError(err)
		}
		if err == nil && len(idFromPodStore) > 0 {
			pair := &reconcilePair{
				resID:   resID,
				resType: deduper.TypePod.String(),
			}
			ret = append(ret, pair)
		}
		pod := reconciliationStore.Get(resID)
		log.Infof("%+v", pod)
		if pod == nil {
			return ret, errList.ToError()
		}
		deploymentID := reconciliationStore.Get(resID).GetDeploymentID()
		idFromDepStore, err := depStore.ReconcileDelete(resType, deploymentID, resHash)
		if err != nil {
			errList.AddError(err)
		}
		if err == nil && len(idFromDepStore) > 0 && len(idFromPodStore) > 0 {
			pair := &reconcilePair{
				resType: deduper.TypeDeployment.String(),
				resID:   deploymentID,
			}
			ret = append(ret, pair)
		}
		return ret, errList.ToError()
	}
}
