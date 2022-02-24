package env

// These environment variables are used in the deployment file
// Please check the files before deleting
var (
	// CentralEndpoint is used to provide Central's reachable endpoint to a sensor.
	CentralEndpoint = RegisterSetting("ROX_CENTRAL_ENDPOINT", WithDefault("central.stackrox.svc:443"))
	// AdvertisedEndpoint is used to provide the Sensor with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = RegisterSetting("ROX_ADVERTISED_ENDPOINT", WithDefault("sensor.stackrox.svc:443"))

	// SensorEndpoint is used to communicate the sensor endpoint to other services in the same cluster.
	SensorEndpoint = RegisterSetting("ROX_SENSOR_ENDPOINT", WithDefault("sensor.stackrox.svc:443"))
)
