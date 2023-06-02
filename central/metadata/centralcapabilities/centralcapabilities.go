package centralcapabilities

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
)

// GetCentralCapabilities informs what Central Services cannot do in the current configuration.
func GetCentralCapabilities() *v1.CentralServicesCapabilities {
	return &v1.CentralServicesCapabilities{
		CentralScanningCanUseContainerIamRoleForEcr: disabledIfManagedCentral(),
		CentralCanUseCloudBackupIntegrations:        disabledIfManagedCentral(),
		CentralCanDisplayDeclarativeConfigHealth:    disabledIfManagedCentral(),
	}
}

func disabledIfManagedCentral() v1.CentralServicesCapabilities_CapabilityStatus {
	if env.ManagedCentral.BooleanSetting() {
		return v1.CentralServicesCapabilities_CapabilityDisabled
	}
	return v1.CentralServicesCapabilities_CapabilityAvailable
}
