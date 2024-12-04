package features

// SensorPullSecretsByName when set to enabled will cause Sensor to capture pull secrets by secret name and registry host instead of just
// registry host.
var SensorPullSecretsByName = registerFeature("Sensor will capture pull secrets by name and registry host instead of just registry host", "ROX_SENSOR_PULL_SECRETS_BY_NAME", enabled)
