package certrefresh

import "github.com/stackrox/rox/pkg/env"

// ROX_SENSOR_CA_ROTATION_ENABLED enables Sensor to advertise CA rotation support to Central.
// This does not gate retrieval or persistence of CA bundles; it only affects the capability set.
// Having the capability set will cause Central to issue Secured Cluster certificates signed by the
// newer CA, if the Central CA has been rotated.
// TODO: Remove when epic ROX-20262 is complete.
var sensorCARotationEnabled = env.RegisterBooleanSetting("ROX_SENSOR_CA_ROTATION_ENABLED", false)
