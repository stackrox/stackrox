package resources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/generated/storage"
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
func InitializeStore() *StoreProvider {
	memSizeSetting := pastEndpointsMemorySize.IntegerSetting()
	if memSizeSetting < 0 {
		memSizeSetting = pastEndpointsMemorySize.DefaultValue()
	}
	log.Infof("Initializing cluster entities store with memory that will last for %d ticks", memSizeSetting)
	deployStore := newDeploymentStore()
	podStore := newPodStore()
	svcStore := newServiceStore()
	nodeStore := newNodeStore()
	entityStore := clusterentities.NewStoreWithMemory(uint16(memSizeSetting))
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

	// FIXME: Conditional start
	p.startDebugServer()

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

func (p *StoreProvider) startDebugServer() *http.Server {
	handler := http.NewServeMux()

	handler.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := make(map[string]int)
		stats["deployments"] = len(p.deploymentStore.GetAll())
		stats["pods"] = len(p.podStore.GetAll())
		stats["services"] = p.serviceStore.getServiceCount()
		stats["networkpolicies"] = len(p.networkPolicyStore.All())
		stats["rbac.roles"] = len(p.rbacStore.GetAllRoles())
		stats["rbac.bindings"] = len(p.rbacStore.GetAllBindings())
		stats["serviceaccounts"] = len(p.serviceAccountStore.GetAllServiceAccountIDs())
		stats["nodes"] = len(p.nodeStore.getNodes())
		stats["orchestratornamespaces"] = len(p.orchestratorNamespaces.All())
		stats["imagesecrets"] = len(p.registryStore.GetAllSecretIDs())

		mar, err := json.Marshal(stats)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		d := make(map[string]interface{})

		d["time_created"] = time.Now().UTC().String()
		d["deployments"] = formatDeployments(p.deploymentStore.GetAll())
		d["pods"] = formatPods(p.podStore.GetAll())
		d["nodes"] = formatNodes(p.nodeStore.GetNodes())
		d["secrets"] = p.registryStore.GetAllSecretIDs()
		d["serviceaccounts"] = p.serviceAccountStore.GetAllServiceAccountIDs()

		roles := p.rbacStore.GetAllRoles()
		var roleRefs []string
		for _, role := range roles {
			roleRefs = append(roleRefs, role.UID())
		}
		d["rbacroles"] = roleRefs

		bindings := p.rbacStore.GetAllBindings()
		var bindingRefs []string
		for _, binding := range bindings {
			bindingRefs = append(bindingRefs, binding.GetID())
		}
		d["rbacbindings"] = bindingRefs

		mar, err := json.Marshal(d)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/deployments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mar, err := json.Marshal(formatDeployments(p.deploymentStore.GetAll()))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/pods", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mar, err := json.Marshal(formatPods(p.podStore.GetAll()))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mar, err := json.Marshal(formatNodes(p.nodeStore.GetNodes()))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/secrets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mar, err := json.Marshal(p.registryStore.GetAllSecretIDs())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/serviceaccounts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mar, err := json.Marshal(p.serviceAccountStore.GetAllServiceAccountIDs())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/rbacroles", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		roles := p.rbacStore.GetAllRoles()
		var roleRefs []string
		for _, role := range roles {
			roleRefs = append(roleRefs, role.UID())
		}
		mar, err := json.Marshal(roleRefs)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler.HandleFunc("/debug/rbacbindings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bindings := p.rbacStore.GetAllBindings()
		var bindingRefs []string
		for _, binding := range bindings {
			bindingRefs = append(bindingRefs, binding.GetID())
		}
		mar, err := json.Marshal(bindingRefs)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(mar)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{Addr: "0.0.0.0:6066", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Warnf("Closing debugging server: %v", err)
		}
	}()
	return srv
}

func formatPods(pods []*storage.Pod) map[string]string {
	result := make(map[string]string, len(pods))
	for _, p := range pods {
		k := fmt.Sprintf("%s/%s", p.GetNamespace(), p.GetName())
		result[k] = p.GetId()
	}
	return result
}

func formatNodes(wraps map[string]*nodeWrap) map[string]string {
	result := make(map[string]string, len(wraps))
	for k, w := range wraps {
		//nolint:gosimple
		result[k] = fmt.Sprintf("%s", w.ObjectMeta.UID)
	}
	return result
}

func formatDeployments(wraps []*storage.Deployment) map[string]string {
	result := make(map[string]string, len(wraps))
	for _, w := range wraps {
		k := fmt.Sprintf("%s/%s", w.GetNamespace(), w.GetName())
		result[k] = w.GetId()
	}
	return result
}
