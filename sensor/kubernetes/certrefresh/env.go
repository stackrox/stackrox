package certrefresh

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/installmethod"
)

// ROX_SENSOR_CA_ROTATION_OPERATOR_ENABLED enables CA rotation support for Operator-managed Sensors.
// When enabled, Operator-managed Sensors will advertise the SensorCARotationSupported capability to Central,
// causing Central to issue Secured Cluster certificates signed by the newer CA during CA rotation.
var sensorCARotationOperatorEnabled = env.RegisterBooleanSetting("ROX_SENSOR_CA_ROTATION_OPERATOR_ENABLED", true)

// ROX_SENSOR_CA_ROTATION_HELM_ENABLED enables CA rotation support for Helm-managed Sensors.
// When enabled, Helm-managed Sensors will advertise the SensorCARotationSupported capability to Central,
// causing Central to issue Secured Cluster certificates signed by the newer CA during CA rotation.
var sensorCARotationHelmEnabled = env.RegisterBooleanSetting("ROX_SENSOR_CA_ROTATION_HELM_ENABLED", false)

// SensorCARotationEnabled returns whether CA rotation capabilities should be enabled for this Sensor,
// based on the installation method and corresponding environment variable settings.
// Returns true if CA rotation is enabled for the current install method, false otherwise.
func SensorCARotationEnabled() bool {
	switch installmethod.Get() {
	case "operator":
		return sensorCARotationOperatorEnabled.BooleanSetting()
	case "helm":
		return sensorCARotationHelmEnabled.BooleanSetting()
	default:
		// Manual/manifest installations do not support CA rotation
		return false
	}
}
