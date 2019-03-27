package resources

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type portRef struct {
	Port     intstr.IntOrString
	Protocol v1.Protocol
}

func portRefOf(svcPort v1.ServicePort) portRef {
	return portRef{
		Port:     svcPort.TargetPort,
		Protocol: svcPort.Protocol,
	}
}
