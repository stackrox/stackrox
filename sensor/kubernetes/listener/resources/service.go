package resources

import (
	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type serviceWithRoutes struct {
	*serviceWrap
	routes []*routeV1.Route
}

type serviceWrap struct {
	*v1.Service
	selector selector.Selector
}

func wrapService(svc *v1.Service) *serviceWrap {
	return &serviceWrap{
		Service:  svc,
		selector: selector.CreateSelector(svc.Spec.Selector, selector.EmptyMatchesNothing()),
	}
}

// getPortMatchFunc takes a target port specified in a route, and returns a function that takes in a
// service port, and returns whether the route is targeting that port of the service or not.
func getPortMatchFunc(port intstr.IntOrString) func(servicePort *v1.ServicePort) bool {
	if port.Type == intstr.String {
		return func(servicePort *v1.ServicePort) bool {
			return servicePort.Name == port.StrVal
		}
	}
	return func(servicePort *v1.ServicePort) bool {
		return servicePort.Port == port.IntVal
	}
}

func exposureInfoFromPort(template *storage.PortConfig_ExposureInfo, port v1.ServicePort) *storage.PortConfig_ExposureInfo {
	out := template.Clone()
	out.ServicePort = port.Port
	out.NodePort = port.NodePort
	return out
}

func (s *serviceWithRoutes) exposure() map[service.PortRef][]*storage.PortConfig_ExposureInfo {
	if s.Spec.Type == v1.ServiceTypeExternalName {
		return nil
	}

	exposureTemplate := &storage.PortConfig_ExposureInfo{
		Level:            storage.PortConfig_INTERNAL,
		ServiceId:        string(s.UID),
		ServiceName:      s.Name,
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

	result := make(map[service.PortRef][]*storage.PortConfig_ExposureInfo, len(s.Spec.Ports))
	for _, port := range s.Spec.Ports {
		ref := service.PortRefOf(port)
		exposureInfo := exposureInfoFromPort(exposureTemplate, port)
		result[ref] = append(result[ref], exposureInfo)
	}

	for _, route := range s.routes {
		routeExposureTemplate := &storage.PortConfig_ExposureInfo{
			Level:            storage.PortConfig_ROUTE,
			ServiceId:        string(s.UID),
			ServiceName:      s.Name,
			ServiceClusterIp: s.Spec.ClusterIP,
		}
		for _, ingress := range route.Status.Ingress {
			if ingress.Host != "" {
				routeExposureTemplate.ExternalHostnames = append(routeExposureTemplate.ExternalHostnames, ingress.Host)
			}
		}
		// if route.Spec.Port is specified, then the route targets one specific port on the service.
		if routePort := route.Spec.Port; routePort != nil {
			matchFunc := getPortMatchFunc(routePort.TargetPort)
			for i, port := range s.Spec.Ports {
				if !matchFunc(&s.Spec.Ports[i]) {
					continue
				}
				ref := service.PortRefOf(port)
				exposureInfo := exposureInfoFromPort(routeExposureTemplate, port)
				result[ref] = append(result[ref], exposureInfo)
				break // Only one port will ever match
			}
		} else {
			// This is the case where route.Spec.Port is not specified, in which case
			// the route targets all ports on the service.
			for _, port := range s.Spec.Ports {
				ref := service.PortRefOf(port)
				exposureInfo := exposureInfoFromPort(routeExposureTemplate, port)
				result[ref] = append(result[ref], exposureInfo)
			}
		}
	}

	return result
}

// serviceDispatcher handles service resource events.
type serviceDispatcher struct {
	serviceStore           *serviceStore
	deploymentStore        *DeploymentStore
	endpointManager        endpointManager
	portExposureReconciler portExposureReconciler
}

// newServiceDispatcher creates and returns a new service handler.
func newServiceDispatcher(serviceStore *serviceStore, deploymentStore *DeploymentStore, endpointManager endpointManager, portExposureReconciler portExposureReconciler) *serviceDispatcher {
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
	oldWrap := sh.serviceStore.getService(svc.Namespace, svc.Name)
	if oldWrap != nil {
		sel = oldWrap.selector
	}
	if action == central.ResourceAction_UPDATE_RESOURCE || action == central.ResourceAction_SYNC_RESOURCE {
		newWrap := wrapService(svc)
		sh.serviceStore.addOrUpdateService(newWrap)
		if sel != nil {
			sel = selector.Or(sel, newWrap.selector)
		} else {
			sel = newWrap.selector
		}
	} else if action == central.ResourceAction_REMOVE_RESOURCE {
		sh.serviceStore.removeService(svc)
	}
	// If OnNamespaceDelete is called before we need to get the selector from the received object
	if sel == nil {
		wrap := wrapService(svc)
		sel = wrap.selector
	}
	return sh.updateDeploymentsFromStore(svc.Namespace, sel)
}

func (sh *serviceDispatcher) updateDeploymentsFromStore(namespace string, sel selector.Selector) *component.ResourceEvent {
	var message component.ResourceEvent
	message.AddDeploymentReference(resolver.ResolveDeploymentLabels(namespace, sel))
	return &message
}

func (sh *serviceDispatcher) processCreate(svc *v1.Service) *component.ResourceEvent {
	svcWrap := wrapService(svc)
	sh.serviceStore.addOrUpdateService(svcWrap)
	var message component.ResourceEvent
	message.AddDeploymentReference(resolver.ResolveDeploymentLabels(svc.GetNamespace(), svcWrap.selector))
	return &message
}
