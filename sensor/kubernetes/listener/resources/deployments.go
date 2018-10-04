package resources

import (
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/listeners"
	"k8s.io/client-go/listers/core/v1"
)

// deploymentHandler handles deployment resource events.
type deploymentHandler struct {
	podLister       v1.PodLister
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	endpointManager *endpointManager
}

// newDeploymentHandler creates and returns a new deployment handler.
func newDeploymentHandler(serviceStore *serviceStore, deploymentStore *deploymentStore, endpointManager *endpointManager, podLister v1.PodLister) *deploymentHandler {
	return &deploymentHandler{
		podLister:       podLister,
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		endpointManager: endpointManager,
	}
}

// Process processes a deployment resource events, and returns the sensor events to emit in response.
func (d *deploymentHandler) Process(obj interface{}, action pkgV1.ResourceAction, deploymentType string) []*listeners.EventWrap {
	wrap := newDeploymentEventFromResource(obj, action, deploymentType, d.podLister)
	if wrap == nil {
		return nil
	}
	wrap.updatePortExposureFromStore(d.serviceStore)
	if action != pkgV1.ResourceAction_REMOVE_RESOURCE {
		d.deploymentStore.addOrUpdateDeployment(wrap)
		d.endpointManager.OnDeploymentCreateOrUpdate(wrap)
	} else {
		d.deploymentStore.removeDeployment(wrap)
		d.endpointManager.OnDeploymentRemove(wrap)
	}

	return []*listeners.EventWrap{{
		SensorEvent: &pkgV1.SensorEvent{
			Id:     wrap.GetId(),
			Action: action,
			Resource: &pkgV1.SensorEvent_Deployment{
				Deployment: wrap.Deployment,
			},
		},
		OriginalSpec: obj,
	}}
}
