package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/roxmetadata"
	"k8s.io/api/core/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
)

// deploymentDispatcherImpl is a Dispatcher implementation for deployment events.
// All deploymentDispatcherImpl must share a handler instance since different types must be correlated.
type deploymentDispatcherImpl struct {
	deploymentType string

	handler *deploymentHandler
}

// newDeploymentDispatcher creates and returns a new deployment dispatcher instance.
func newDeploymentDispatcher(deploymentType string, handler *deploymentHandler) Dispatcher {
	return &deploymentDispatcherImpl{
		deploymentType: deploymentType,
		handler:        handler,
	}
}

// ProcessEvent processes a deployment resource events, and returns the sensor events to emit in response.
func (d *deploymentDispatcherImpl) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	return d.handler.processWithType(obj, action, d.deploymentType)
}

// deploymentHandler handles deployment resource events and does the actual processing.
type deploymentHandler struct {
	podLister       v1listers.PodLister
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	endpointManager *endpointManager
	namespaceStore  *namespaceStore
	roxMetadata     roxmetadata.Metadata
}

// newDeploymentHandler creates and returns a new deployment handler.
func newDeploymentHandler(serviceStore *serviceStore, deploymentStore *deploymentStore, endpointManager *endpointManager, namespaceStore *namespaceStore, roxMetadata roxmetadata.Metadata, podLister v1listers.PodLister) *deploymentHandler {
	return &deploymentHandler{
		podLister:       podLister,
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		endpointManager: endpointManager,
		namespaceStore:  namespaceStore,
		roxMetadata:     roxMetadata,
	}
}

func (d *deploymentHandler) processWithType(obj interface{}, action central.ResourceAction, deploymentType string) []*central.SensorEvent {
	wrap := newDeploymentEventFromResource(obj, action, deploymentType, d.podLister, d.namespaceStore)
	if wrap == nil {
		return d.maybeProcessPod(obj)
	}
	wrap.updatePortExposureFromStore(d.serviceStore)
	if action != central.ResourceAction_REMOVE_RESOURCE {
		d.deploymentStore.addOrUpdateDeployment(wrap)
		d.endpointManager.OnDeploymentCreateOrUpdate(wrap)
		d.roxMetadata.AddDeployment(wrap.GetDeployment())
	} else {
		d.deploymentStore.removeDeployment(wrap)
		d.endpointManager.OnDeploymentRemove(wrap)
	}
	return []*central.SensorEvent{wrap.toEvent(action)}
}

func (d *deploymentHandler) maybeProcessPod(obj interface{}) []*central.SensorEvent {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return nil
	}
	owners := d.deploymentStore.getOwningDeployments(pod.Namespace, pod.Labels)
	var events []*central.SensorEvent
	for _, owner := range owners {
		events = append(events, d.processWithType(owner.original, central.ResourceAction_UPDATE_RESOURCE, owner.Type)...)
	}
	return events
}
