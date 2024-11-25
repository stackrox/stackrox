package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// SensorCapturesIntermediateEvents enables sensor to capture intermediate events when it is disconnected from central
var SensorCapturesIntermediateEvents = registerFeature("Enables sensor to capture intermediate events when it is disconnected from central", "ROX_CAPTURE_INTERMEDIATE_EVENTS", enabled)
