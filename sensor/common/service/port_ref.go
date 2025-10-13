package service

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// PortRef is the reference to a service's port.
type PortRef struct {
	Port     intstr.IntOrString
	Protocol v1.Protocol
}

// PortRefOf returns a PortRef definition based on a v1.ServicePort spec.
func PortRefOf(svcPort v1.ServicePort) PortRef {
	return PortRef{
		Port:     svcPort.TargetPort,
		Protocol: svcPort.Protocol,
	}
}
