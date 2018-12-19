package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"k8s.io/api/core/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
)

// deploymentHandler handles deployment resource events.
type deploymentHandler struct {
	podLister       v1listers.PodLister
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	endpointManager *endpointManager
}

// newDeploymentHandler creates and returns a new deployment handler.
func newDeploymentHandler(serviceStore *serviceStore, deploymentStore *deploymentStore, endpointManager *endpointManager, podLister v1listers.PodLister) *deploymentHandler {
	return &deploymentHandler{
		podLister:       podLister,
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		endpointManager: endpointManager,
	}
}

func (d *deploymentHandler) maybeProcessPod(obj interface{}) []*central.SensorEvent {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return nil
	}
	owners := d.deploymentStore.getOwningDeployments(pod.Namespace, pod.Labels)
	var events []*central.SensorEvent
	for _, owner := range owners {
		events = append(events, d.Process(owner.original, central.ResourceAction_UPDATE_RESOURCE, owner.Type)...)
	}
	return events
}

// Process processes a deployment resource events, and returns the sensor events to emit in response.
func (d *deploymentHandler) Process(obj interface{}, action central.ResourceAction, deploymentType string) []*central.SensorEvent {
	wrap := newDeploymentEventFromResource(obj, action, deploymentType, d.podLister)
	if wrap == nil {
		return d.maybeProcessPod(obj)
	}
	wrap.updatePortExposureFromStore(d.serviceStore)
	if action != central.ResourceAction_REMOVE_RESOURCE {
		d.deploymentStore.addOrUpdateDeployment(wrap)
		d.endpointManager.OnDeploymentCreateOrUpdate(wrap)
	} else {
		d.deploymentStore.removeDeployment(wrap)
		d.endpointManager.OnDeploymentRemove(wrap)
	}

	return []*central.SensorEvent{wrap.toEvent(action)}
}
