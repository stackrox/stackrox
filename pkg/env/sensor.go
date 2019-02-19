package env

// These environment variables are used in the deployment file
// Please check the files before deleting
var (
	// ClusterID is used to provide a cluster ID to a sensor.
	// This cluster ID is not relied upon for authentication or authorization.
	ClusterID = NewSetting("ROX_CLUSTER_ID")
	// CentralEndpoint is used to provide Central's reachable endpoint to a sensor.
	CentralEndpoint = NewSetting("ROX_CENTRAL_ENDPOINT", WithDefault("central.prevent_net:443"))
	// AdvertisedEndpoint is used to provide the Sensor with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = NewSetting("ROX_ADVERTISED_ENDPOINT", WithDefault("sensor.prevent_net:443"))
	// Image is the image that should be launched for new benchmarks.
	// Deprecated: This will go away with the old benchmarks code.
	Image = NewSetting("ROX_MAIN_IMAGE")
)
