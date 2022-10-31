package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/store/service/servicewrapper"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/core/v1"
)

// serviceDispatcher handles service resource events.
type serviceDispatcher struct {
	serviceStore           store.ServiceStore
	deploymentStore        *DeploymentStore
	endpointManager        endpointManager
	portExposureReconciler portExposureReconciler
}

// newServiceDispatcher creates and returns a new service handler.
func newServiceDispatcher(serviceStore store.ServiceStore, deploymentStore *DeploymentStore, endpointManager endpointManager, portExposureReconciler portExposureReconciler) *serviceDispatcher {
	return &serviceDispatcher{
		serviceStore:           serviceStore,
		deploymentStore:        deploymentStore,
		endpointManager:        endpointManager,
		portExposureReconciler: portExposureReconciler,
	}
}

// ProcessEvent processes a service resource event, and returns the sensor events to emit in response.
func (sh *serviceDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	svc := obj.(*v1.Service)
	if action == central.ResourceAction_CREATE_RESOURCE {
		return sh.processCreate(svc)
	}
	var sel selector.Selector
	oldWrap := sh.serviceStore.GetService(svc.Namespace, svc.Name)
	if oldWrap != nil {
		sel = oldWrap.Selector
	}
	if action == central.ResourceAction_UPDATE_RESOURCE || action == central.ResourceAction_SYNC_RESOURCE {
		newWrap := servicewrapper.WrapService(svc)
		sh.serviceStore.UpsertService(newWrap)
		if sel != nil {
			sel = selector.Or(sel, newWrap.Selector)
		} else {
			sel = newWrap.Selector
		}
	} else if action == central.ResourceAction_REMOVE_RESOURCE {
		sh.serviceStore.RemoveService(svc)
	}
	// If OnNamespaceDelete is called before we need to get the selector from the received object
	if sel == nil {
		wrap := servicewrapper.WrapService(svc)
		sel = wrap.Selector
	}
	return sh.updateDeploymentsFromStore(svc.Namespace, sel)
}

func (sh *serviceDispatcher) updateDeploymentsFromStore(namespace string, sel selector.Selector) *component.ResourceEvent {
	events := sh.portExposureReconciler.UpdateExposuresForMatchingDeployments(namespace, sel)
	sh.endpointManager.OnServiceUpdateOrRemove(namespace, sel)
	return component.NewResourceEvent(events, nil, nil)
}

func (sh *serviceDispatcher) processCreate(svc *v1.Service) *component.ResourceEvent {
	svcWrap := servicewrapper.WrapService(svc)
	sh.serviceStore.UpsertService(svcWrap)
	events := sh.portExposureReconciler.UpdateExposureOnServiceCreate(servicewrapper.SelectorRouteWrap{
		SelectorWrap: svcWrap,
		Routes:       sh.serviceStore.GetRoutesForService(svcWrap),
	})
	sh.endpointManager.OnServiceCreate(svcWrap)
	return component.NewResourceEvent(events, nil, nil)
}
