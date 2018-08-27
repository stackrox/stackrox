package idcheck

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// SensorsOnly returns a serviceType authorizer that checks for the Sensor type.
func SensorsOnly() authz.Authorizer {
	return serviceType{Type: v1.ServiceType_SENSOR_SERVICE}
}
