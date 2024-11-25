package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// PreventSensorRestartOnDisconnect enables a new behavior in Sensor where it avoids restarting when the gRPC connection with Central ends.
var PreventSensorRestartOnDisconnect = registerFeature("Prevent Sensor restart on disconnect", "ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", enabled, unchangeableInProd)
