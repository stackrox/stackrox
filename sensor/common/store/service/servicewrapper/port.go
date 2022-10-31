package servicewrapper

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// PortRef is the reference to an exposed port in a service
type PortRef struct {
	Port     intstr.IntOrString
	Protocol v1.Protocol
}

// PortRefOf returns a PortRef struct from a kubernetes v1.ServicePort
func PortRefOf(svcPort v1.ServicePort) PortRef {
	return PortRef{
		Port:     svcPort.TargetPort,
		Protocol: svcPort.Protocol,
	}
}
