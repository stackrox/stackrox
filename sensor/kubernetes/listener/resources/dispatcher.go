package resources

import (
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	v1Listers "k8s.io/client-go/listers/core/v1"
)

// Dispatcher is responsible for processing resource events, and returning the sensor events that should be emitted
// in response.
type Dispatcher interface {
	ProcessEvent(obj interface{}, action pkgV1.ResourceAction, deploymentType string) []*listeners.EventWrap
}

// NewDispatcher creates and returns a new dispatcher.
func NewDispatcher(podLister v1Listers.PodLister, entityStore *clusterentities.Store) Dispatcher {
	serviceStore := newServiceStore()
	deploymentStore := newDeploymentStore()
	nodeStore := newNodeStore()
	endpointManager := newEndpointManager(serviceStore, deploymentStore, nodeStore, entityStore)

	return &dispatcher{
		namespaceHandler:     newNamespaceHandler(serviceStore, deploymentStore),
		deploymentHandler:    newDeploymentHandler(serviceStore, deploymentStore, endpointManager, podLister),
		serviceHandler:       newServiceHandler(serviceStore, deploymentStore, endpointManager),
		secretHandler:        newSecretHandler(),
		networkPolicyHandler: newNetworkPolicyHandler(),
		nodeHandler:          newNodeHandler(serviceStore, deploymentStore, nodeStore, endpointManager),
	}
}

type dispatcher struct {
	namespaceHandler     *namespaceHandler
	deploymentHandler    *deploymentHandler
	serviceHandler       *serviceHandler
	secretHandler        *secretHandler
	networkPolicyHandler *networkPolicyHandler
	nodeHandler          *nodeHandler
}

func (d *dispatcher) ProcessEvent(obj interface{}, action pkgV1.ResourceAction, deploymentType string) []*listeners.EventWrap {
	if deploymentType != "" {
		return d.deploymentHandler.Process(obj, action, deploymentType)
	}

	switch o := obj.(type) {
	case *v1.Service:
		return d.serviceHandler.Process(o, action)
	case *v1.Secret:
		return d.secretHandler.Process(o, action)
	case *networkingV1.NetworkPolicy:
		return d.networkPolicyHandler.Process(o, action)
	case *v1.Namespace:
		return d.namespaceHandler.Process(o, action)
	case *v1.Node:
		return d.nodeHandler.Process(o, action)
	default:
		return nil
	}
}
