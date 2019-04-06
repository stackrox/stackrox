package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/roxmetadata"
	v1Listers "k8s.io/client-go/listers/core/v1"
)

// Dispatcher is responsible for processing resource events, and returning the sensor events that should be emitted
// in response.
//go:generate mockgen-wrapper Dispatcher
type Dispatcher interface {
	ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent
}

// DispatcherRegistry provides dispatchers to use.
type DispatcherRegistry interface {
	ForDeployments(deploymentType string) Dispatcher

	ForNamespaces() Dispatcher
	ForNetworkPolicies() Dispatcher
	ForNodes() Dispatcher
	ForSecrets() Dispatcher
	ForServices() Dispatcher
	ForServiceAccounts() Dispatcher
	ForRoles() Dispatcher
	ForClusterRoles() Dispatcher
	ForRoleBindings() Dispatcher
	ForClusterRoleBindings() Dispatcher
}

// NewDispatcherRegistry creates and returns a new DispatcherRegistry.
func NewDispatcherRegistry(podLister v1Listers.PodLister, entityStore *clusterentities.Store, roxMetadata roxmetadata.Metadata) DispatcherRegistry {
	serviceStore := newServiceStore()
	deploymentStore := newDeploymentStore()
	nodeStore := newNodeStore()
	nsStore := newNamespaceStore()
	endpointManager := newEndpointManager(serviceStore, deploymentStore, nodeStore, entityStore)
	rbacStore := newRBACStore()

	return &registryImpl{
		deploymentHandler: newDeploymentHandler(serviceStore, deploymentStore, endpointManager, nsStore, roxMetadata, podLister),
		rbacHandler:       newRBACHandler(rbacStore),

		namespaceDispatcher:      newNamespaceDispatcher(nsStore, serviceStore, deploymentStore),
		serviceDispatcher:        newServiceDispatcher(serviceStore, deploymentStore, endpointManager),
		secretDispatcher:         newSecretDispatcher(),
		networkPolicyDispatcher:  newNetworkPolicyDispatcher(),
		nodeDispatcher:           newNodeDispatcher(serviceStore, deploymentStore, nodeStore, endpointManager),
		serviceAccountDispatcher: newServiceAccountDispatcher(),
	}
}

type registryImpl struct {
	deploymentHandler *deploymentHandler
	rbacHandler       *rbacHandler

	namespaceDispatcher      *namespaceDispatcher
	serviceDispatcher        *serviceDispatcher
	secretDispatcher         *secretDispatcher
	networkPolicyDispatcher  *networkPolicyDispatcher
	nodeDispatcher           *nodeDispatcher
	serviceAccountDispatcher *serviceAccountDispatcher

	registryLock sync.Mutex
}

func (d *registryImpl) ForDeployments(deploymentType string) Dispatcher {
	return d.newExclusive(newDeploymentDispatcher(deploymentType, d.deploymentHandler))
}

func (d *registryImpl) ForNamespaces() Dispatcher {
	return d.newExclusive(d.namespaceDispatcher)
}

func (d *registryImpl) ForNetworkPolicies() Dispatcher {
	return d.newExclusive(d.networkPolicyDispatcher)
}

func (d *registryImpl) ForNodes() Dispatcher {
	return d.newExclusive(d.nodeDispatcher)
}

func (d *registryImpl) ForSecrets() Dispatcher {
	return d.newExclusive(d.secretDispatcher)
}

func (d *registryImpl) ForServices() Dispatcher {
	return d.newExclusive(d.serviceDispatcher)
}

func (d *registryImpl) ForServiceAccounts() Dispatcher {
	return d.newExclusive(d.serviceAccountDispatcher)
}

func (d *registryImpl) ForRoles() Dispatcher {
	return d.newExclusive(newRoleDispatcher(d.rbacHandler))
}

func (d *registryImpl) ForClusterRoles() Dispatcher {
	return d.newExclusive(newClusterRoleDispatcher(d.rbacHandler))
}

func (d *registryImpl) ForRoleBindings() Dispatcher {
	return d.newExclusive(newRoleBindingDispatcher(d.rbacHandler))
}

func (d *registryImpl) ForClusterRoleBindings() Dispatcher {
	return d.newExclusive(newClusterRoleBindingDispatcher(d.rbacHandler))
}

// Wraps the input dispatcher so that only a single dispatcher returned from the registry can be run at a time.
func (d *registryImpl) newExclusive(dispatcher Dispatcher) Dispatcher {
	return &exclusiveDispatcherImpl{
		dispatcher: dispatcher,
		lock:       &d.registryLock,
	}
}

// Helper class that wraps a Dispatcher to make a set of Dispatchers run mutually exclusive.
type exclusiveDispatcherImpl struct {
	dispatcher Dispatcher
	lock       *sync.Mutex
}

func (e *exclusiveDispatcherImpl) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.dispatcher.ProcessEvent(obj, action)
}
