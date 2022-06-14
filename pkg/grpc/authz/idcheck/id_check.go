package idcheck

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// SensorsOnly returns a serviceType authorizer that checks for the Sensor type.
func SensorsOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SENSOR_SERVICE))
}

// ScannerOnly returns a serviceType authorizer that checks for the scanner type.
func ScannerOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SCANNER_SERVICE))
}

// CollectorOnly returns a serviceType authorizer that checks for the Collector type.
func CollectorOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_COLLECTOR_SERVICE))
}

// AdmissionControlOnly returns an authorizer that checks for the Admission Control type.
func AdmissionControlOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_ADMISSION_CONTROL_SERVICE))
}
