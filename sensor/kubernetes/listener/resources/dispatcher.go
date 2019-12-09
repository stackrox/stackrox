package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/config"
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
	ForRoleBindings() Dispatcher
	ForClusterRoles() Dispatcher
	ForClusterRoleBindings() Dispatcher
}

// NewDispatcherRegistry creates and returns a new DispatcherRegistry.
func NewDispatcherRegistry(podLister v1Listers.PodLister, entityStore *clusterentities.Store, roxMetadata roxmetadata.Metadata, processFilter filter.Filter, configHandler config.Handler) DispatcherRegistry {
	serviceStore := newServiceStore()
	deploymentStore := newDeploymentStore()
	nodeStore := newNodeStore()
	nsStore := newNamespaceStore()
	endpointManager := newEndpointManager(serviceStore, deploymentStore, nodeStore, entityStore)
	rbacUpdater := newRBACUpdater()

	return &registryImpl{
		deploymentHandler: newDeploymentHandler(serviceStore, deploymentStore, endpointManager, nsStore, roxMetadata, podLister, processFilter, configHandler),

		roleDispatcher:           newRoleDispatcher(rbacUpdater),
		bindingDispatcher:        newBindingDispatcher(rbacUpdater),
		clusterRoleDispatcher:    newClusterRoleDispatcher(rbacUpdater),
		clusterBindingDispatcher: newClusterBindingDispatcher(rbacUpdater),
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

	roleDispatcher           *roleDispatcher
	bindingDispatcher        *bindingDispatcher
	clusterRoleDispatcher    *clusterRoleDispatcher
	clusterBindingDispatcher *clusterBindingDispatcher
	namespaceDispatcher      *namespaceDispatcher
	serviceDispatcher        *serviceDispatcher
	secretDispatcher         *secretDispatcher
	networkPolicyDispatcher  *networkPolicyDispatcher
	nodeDispatcher           *nodeDispatcher
	serviceAccountDispatcher *serviceAccountDispatcher
}

func (d *registryImpl) ForDeployments(deploymentType string) Dispatcher {
	return newDeploymentDispatcher(deploymentType, d.deploymentHandler)
}

func (d *registryImpl) ForNamespaces() Dispatcher {
	return d.namespaceDispatcher
}

func (d *registryImpl) ForNetworkPolicies() Dispatcher {
	return d.networkPolicyDispatcher
}

func (d *registryImpl) ForNodes() Dispatcher {
	return d.nodeDispatcher
}

func (d *registryImpl) ForSecrets() Dispatcher {
	return d.secretDispatcher
}

func (d *registryImpl) ForServices() Dispatcher {
	return d.serviceDispatcher
}

func (d *registryImpl) ForServiceAccounts() Dispatcher {
	return d.serviceAccountDispatcher
}

func (d *registryImpl) ForRoles() Dispatcher {
	return d.roleDispatcher
}

func (d *registryImpl) ForRoleBindings() Dispatcher {
	return d.bindingDispatcher
}

func (d *registryImpl) ForClusterRoles() Dispatcher {
	return d.clusterRoleDispatcher
}

func (d *registryImpl) ForClusterRoleBindings() Dispatcher {
	return d.clusterBindingDispatcher
}
