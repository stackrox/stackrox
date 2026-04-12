package env

// SensorLite enables lightweight sensor mode for edge deployments.
// When enabled:
// - Policy evaluation is deferred to central (sensor forwards raw events)
// - Network entity knowledge base is not loaded
// - Informer caches are minimized
// This significantly reduces memory and CPU usage at the cost of
// slightly delayed alerts (~200ms round-trip to central).
var SensorLite = RegisterBooleanSetting("ROX_SENSOR_LITE", false)
