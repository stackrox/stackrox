package centralcapabilities

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

// GetCentralCapabilities informs what Central Services cannot do in the current configuration.
func GetCentralCapabilities() *v1.CentralServicesCapabilities {
	csc := &v1.CentralServicesCapabilities{}
	csc.SetCentralScanningCanUseContainerIamRoleForEcr(disabledIfManagedCentral())
	csc.SetCentralCanUseCloudBackupIntegrations(disabledIfExternalDatabase())
	csc.SetCentralCanDisplayDeclarativeConfigHealth(disabledIfManagedCentral())
	csc.SetCentralCanUpdateCert(disabledIfManagedCentral())
	csc.SetCentralCanUseAcscsEmailIntegration(enabledIfManagedCentral())
	return csc
}

func enabledIfManagedCentral() v1.CentralServicesCapabilities_CapabilityStatus {
	if env.ManagedCentral.BooleanSetting() {
		return v1.CentralServicesCapabilities_CapabilityAvailable
	}

	return v1.CentralServicesCapabilities_CapabilityDisabled
}

func disabledIfManagedCentral() v1.CentralServicesCapabilities_CapabilityStatus {
	if env.ManagedCentral.BooleanSetting() {
		return v1.CentralServicesCapabilities_CapabilityDisabled
	}
	return v1.CentralServicesCapabilities_CapabilityAvailable
}

func disabledIfExternalDatabase() v1.CentralServicesCapabilities_CapabilityStatus {
	if env.ManagedCentral.BooleanSetting() || pgconfig.IsExternalDatabase() {
		return v1.CentralServicesCapabilities_CapabilityDisabled
	}
	return v1.CentralServicesCapabilities_CapabilityAvailable
}
