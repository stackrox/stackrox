package features

// PreventSensorRestartOnDisconnect enables a new behavior in Sensor where it avoids restarting when the gRPC connection with Central ends.
var PreventSensorRestartOnDisconnect = registerFeature("Prevent Sensor restart on disconnect", "ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", enabled, unchangeableInProd)
