package env

// These environment variables are used in the deployment file
// Please check the files before deleting
var (
	// ClusterID is used to provide a cluster ID to a sensor.
	// This cluster ID is not relied upon for authentication or authorization.
	ClusterID = RegisterSetting("ROX_CLUSTER_ID")
	// CentralEndpoint is used to provide Central's reachable endpoint to a sensor.
	CentralEndpoint = RegisterSetting("ROX_CENTRAL_ENDPOINT", WithDefault("central.stackrox:443"))
	// AdvertisedEndpoint is used to provide the Sensor with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = RegisterSetting("ROX_ADVERTISED_ENDPOINT", WithDefault("sensor.stackrox:443"))
)
