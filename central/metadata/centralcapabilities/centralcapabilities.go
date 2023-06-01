package centralcapabilities

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
)

// GetCentralCapabilities informs what Central Services cannot do in the current configuration.
func GetCentralCapabilities() *v1.CentralServicesCapabilities {
	return &v1.CentralServicesCapabilities{
		CentralScanningUseContainerIamRoleForEcr: disabledIfManagedCentral(),
		CentralCloudBackupIntegrations:           disabledIfManagedCentral(),
	}
}

func disabledIfManagedCentral() v1.CentralServicesCapabilities_CapabilityStatus {
	if env.ManagedCentral.BooleanSetting() {
		return v1.CentralServicesCapabilities_Disabled
	}
	return v1.CentralServicesCapabilities_Unknown
}
