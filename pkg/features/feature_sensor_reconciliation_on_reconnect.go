package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// SensorReconciliationOnReconnect enables sensors to support reconciliation when reconnecting
var SensorReconciliationOnReconnect = registerFeature("Enable Sensors to support reconciliation on reconnect", "ROX_SENSOR_RECONCILIATION", enabled)
