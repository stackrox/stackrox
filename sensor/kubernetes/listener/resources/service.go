package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/core/v1"
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

func (s *serviceWrap) exposure() map[portRef]*storage.PortConfig_ExposureInfo {
	if s.Spec.Type == v1.ServiceTypeExternalName {
		return nil
	}

	exposureTemplate := storage.PortConfig_ExposureInfo{
		Level:            storage.PortConfig_INTERNAL,
		ServiceId:        string(s.UID),
		ServiceName:      string(s.Name),
		ServiceClusterIp: s.Spec.ClusterIP,
	}

	if s.Spec.Type == v1.ServiceTypeNodePort {
		exposureTemplate.Level = storage.PortConfig_NODE
	} else if s.Spec.Type == v1.ServiceTypeLoadBalancer {
		exposureTemplate.Level = storage.PortConfig_EXTERNAL
		for _, lbIngress := range s.Status.LoadBalancer.Ingress {
			if lbIngress.IP != "" {
				exposureTemplate.ExternalIps = append(exposureTemplate.ExternalIps, lbIngress.IP)
			}
			if lbIngress.Hostname != "" {
				exposureTemplate.ExternalHostnames = append(exposureTemplate.ExternalHostnames, lbIngress.Hostname)
			}
		}
	}

	result := make(map[portRef]*storage.PortConfig_ExposureInfo, len(s.Spec.Ports))
	for _, port := range s.Spec.Ports {
		ref := portRefOf(port)
		exposureInfo := exposureTemplate
		exposureInfo.ServicePort = port.Port
		exposureInfo.NodePort = port.NodePort
		result[ref] = &exposureInfo
	}

	return result
}

// serviceDispatcher handles servidce resource events.
type serviceDispatcher struct {
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	endpointManager *endpointManager
}

// newServiceDispatcher creates and returns a new service handler.
func newServiceDispatcher(serviceStore *serviceStore, deploymentStore *deploymentStore, endpointManager *endpointManager) *serviceDispatcher {
	return &serviceDispatcher{
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		endpointManager: endpointManager,
	}
}

// Process processes a service resource event, and returns the sensor events to emit in response.
func (sh *serviceDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	svc := obj.(*v1.Service)
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
	} else if action == central.ResourceAction_REMOVE_RESOURCE {
		sh.serviceStore.removeService(svc)
	}
	return sh.updateDeploymentsFromStore(svc.Namespace, sel)
}

func (sh *serviceDispatcher) updateDeploymentsFromStore(namespace string, sel selector) (events []*central.SensorEvent) {
	for _, deploymentWrap := range sh.deploymentStore.getMatchingDeployments(namespace, sel) {
		deploymentWrap.updatePortExposureFromStore(sh.serviceStore)
		events = append(events, deploymentWrap.toEvent(central.ResourceAction_UPDATE_RESOURCE))
	}
	sh.endpointManager.OnServiceUpdateOrRemove(namespace, sel)
	return
}

func (sh *serviceDispatcher) processCreate(svc *v1.Service) (events []*central.SensorEvent) {
	wrap := wrapService(svc)
	sh.serviceStore.addOrUpdateService(wrap)
	for _, deploymentWrap := range sh.deploymentStore.getMatchingDeployments(svc.Namespace, wrap.selector) {
		deploymentWrap.updatePortExposure(wrap)
		events = append(events, deploymentWrap.toEvent(central.ResourceAction_UPDATE_RESOURCE))
	}
	sh.endpointManager.OnServiceCreate(wrap)
	return
}
