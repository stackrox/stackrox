package servicewrapper

import (
	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/selector"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SelectorRouteWrap wraps a service with routes and selectors
type SelectorRouteWrap struct {
	*SelectorWrap
	Routes []*routeV1.Route
}

// SelectorWrap wraps a service with selectors
type SelectorWrap struct {
	*v1.Service
	Selector selector.Selector
}

// WrapService returns a service object with selector objects
func WrapService(svc *v1.Service) *SelectorWrap {
	return &SelectorWrap{
		Service:  svc,
		Selector: selector.CreateSelector(svc.Spec.Selector, selector.EmptyMatchesNothing()),
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

// Exposure returns port exposure info
func (s *SelectorRouteWrap) Exposure() map[PortRef][]*storage.PortConfig_ExposureInfo {
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

	result := make(map[PortRef][]*storage.PortConfig_ExposureInfo, len(s.Spec.Ports))
	for _, port := range s.Spec.Ports {
		ref := PortRefOf(port)
		exposureInfo := exposureInfoFromPort(exposureTemplate, port)
		result[ref] = append(result[ref], exposureInfo)
	}

	for _, route := range s.Routes {
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
				ref := PortRefOf(port)
				exposureInfo := exposureInfoFromPort(routeExposureTemplate, port)
				result[ref] = append(result[ref], exposureInfo)
				break // Only one port will ever match
			}
		} else {
			// This is the case where route.Spec.Port is not specified, in which case
			// the route targets all ports on the service.
			for _, port := range s.Spec.Ports {
				ref := PortRefOf(port)
				exposureInfo := exposureInfoFromPort(routeExposureTemplate, port)
				result[ref] = append(result[ref], exposureInfo)
			}
		}
	}

	return result
}
