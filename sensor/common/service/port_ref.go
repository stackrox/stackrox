package service

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type PortRef struct {
	Port     intstr.IntOrString
	Protocol v1.Protocol
}

func PortRefOf(svcPort v1.ServicePort) PortRef {
	return PortRef{
		Port:     svcPort.TargetPort,
		Protocol: svcPort.Protocol,
	}
}
