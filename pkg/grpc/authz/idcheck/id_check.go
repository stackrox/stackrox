package idcheck

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// SensorsOnly returns a serviceType authorizer that checks for the Sensor type.
func SensorsOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SENSOR_SERVICE))
}

// CollectorOnly returns a serviceType authorizer that checks for the Collector type.
func CollectorOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_COLLECTOR_SERVICE))
}
