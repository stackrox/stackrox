package idcheck

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// CentralOnly returns a serviceType authorizer that checks for the Central type.
func CentralOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_CENTRAL_SERVICE))
}

// SensorsOnly returns a serviceType authorizer that checks for the Sensor type.
func SensorsOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SENSOR_SERVICE))
}

// SensorRegistrantsOnly returns a serviceType authorizer that checks for the Registrant type.
func SensorRegistrantsOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_REGISTRANT_SERVICE))
}

// ScannerOnly returns a serviceType authorizer that checks for the scanner type.
func ScannerOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SCANNER_SERVICE))
}

// ScannerV4IndexerOnly returns a serviceType authorizer that checks for the Scanner v4 Indexer type.
func ScannerV4IndexerOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE))
}

// ScannerV4MatcherOnly returns a serviceType authorizer that checks for the Scanner v4 Matcher type.
func ScannerV4MatcherOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_SCANNER_V4_MATCHER_SERVICE))
}

// CollectorOnly returns a serviceType authorizer that checks for the Collector type.
func CollectorOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_COLLECTOR_SERVICE))
}

// AdmissionControlOnly returns an authorizer that checks for the Admission Control type.
func AdmissionControlOnly() authz.Authorizer {
	return Wrap(serviceType(storage.ServiceType_ADMISSION_CONTROL_SERVICE))
}
