package idcheck

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
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
