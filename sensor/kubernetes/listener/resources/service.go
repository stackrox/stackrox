package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type serviceWrap struct {
	*v1.Service
	selector labels.Selector
}

func wrapService(svc *v1.Service) *serviceWrap {
	return &serviceWrap{
		Service:  svc,
		selector: SelectorFromMap(svc.Spec.Selector),
	}
}

func (s *serviceWrap) exposure() storage.PortConfig_Exposure {
	switch s.Spec.Type {
	case v1.ServiceTypeLoadBalancer:
		return storage.PortConfig_EXTERNAL
	case v1.ServiceTypeNodePort:
		return storage.PortConfig_NODE
	default:
		return storage.PortConfig_INTERNAL
	}
}

// serviceHandler handles servidce resource events.
type serviceHandler struct {
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	endpointManager *endpointManager
}

// newServiceHandler creates and returns a new service handler.
func newServiceHandler(serviceStore *serviceStore, deploymentStore *deploymentStore, endpointManager *endpointManager) *serviceHandler {
	return &serviceHandler{
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		endpointManager: endpointManager,
	}
}

// Process processes a service resource event, and returns the sensor events to emit in response.
func (sh *serviceHandler) Process(svc *v1.Service, action central.ResourceAction) []*central.SensorEvent {
	if action == central.ResourceAction_CREATE_RESOURCE {
		return sh.processCreate(svc)
	}
	var sel selector
	oldWrap := sh.serviceStore.getService(svc.Namespace, svc.UID)
	if oldWrap != nil {
		sel = oldWrap.selector
	}
	if action == central.ResourceAction_UPDATE_RESOURCE {
		newWrap := wrapService(svc)
		sh.serviceStore.addOrUpdateService(newWrap)
		if sel != nil {
			sel = or(sel, newWrap.selector)
		} else {
			sel = newWrap.selector
		}
	}
	return sh.updateDeploymentsFromStore(svc.Namespace, sel)
}

func (sh *serviceHandler) updateDeploymentsFromStore(namespace string, sel selector) (events []*central.SensorEvent) {
	for _, deploymentWrap := range sh.deploymentStore.getMatchingDeployments(namespace, sel) {
		if deploymentWrap.updatePortExposureFromStore(sh.serviceStore) {
			events = append(events, deploymentWrap.toEvent(central.ResourceAction_UPDATE_RESOURCE))
		}
	}
	sh.endpointManager.OnServiceUpdateOrRemove(namespace, sel)
	return
}

func (sh *serviceHandler) processCreate(svc *v1.Service) (events []*central.SensorEvent) {
	wrap := wrapService(svc)
	sh.serviceStore.addOrUpdateService(wrap)
	for _, deploymentWrap := range sh.deploymentStore.getMatchingDeployments(svc.Namespace, wrap.selector) {
		if deploymentWrap.updatePortExposure(wrap) {
			events = append(events, deploymentWrap.toEvent(central.ResourceAction_UPDATE_RESOURCE))
		}
	}
	sh.endpointManager.OnServiceCreate(wrap)
	return
}
