package idcheck

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// SensorsOnly returns a serviceType authorizer that checks for the Sensor type.
func SensorsOnly() authz.Authorizer {
	return Wrap(serviceType(v1.ServiceType_SENSOR_SERVICE))
}

// CollectorOnly returns a serviceType authorizer that checks for the Collector type.
func CollectorOnly() authz.Authorizer {
	return Wrap(serviceType(v1.ServiceType_COLLECTOR_SERVICE))
}
