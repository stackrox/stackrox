package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
)

func getAutoUpgradeFeatureStatus() v1.SensorToggleConfig_SensorAutoUpgradeFeatureStatus {
	if env.ManagedCentral.BooleanSetting() {
		return v1.SensorToggleConfig_NOT_SUPPORTED
	}
	return v1.SensorToggleConfig_SUPPORTED
}
