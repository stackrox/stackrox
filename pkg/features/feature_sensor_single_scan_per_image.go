package features

// SensorSingleScanPerImage when set to enabled forces Sensor to allow only a single scan per image to be active at any given
// time. Will only have an affect if UnqualifiedSearchRegistries is also enabled.
// TODO(ROX-24641): Remove dependency on the UnqualifiedSearchRegistries feature so that this is enabled by default.
var SensorSingleScanPerImage = registerFeature("Sensor will only allow a single active scan per image", "ROX_SENSOR_SINGLE_SCAN", enabled)
