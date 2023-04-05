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

	// ScannerSlimGRPCEndpoint is used to communicate the scanner endpoint to other services in the same cluster.
	// This is typically used for Sensor to communicate with a local Scanner-slim's gRPC server.
	ScannerSlimGRPCEndpoint = RegisterSetting("ROX_SCANNER_GRPC_ENDPOINT", WithDefault("scanner.stackrox.svc:8443"))

	// LocalImageScanningEnabled is used to specify if Sensor should attempt to scan images via a local Scanner.
	LocalImageScanningEnabled = RegisterBooleanSetting("ROX_LOCAL_IMAGE_SCANNING_ENABLED", false)

	// ForceLocalImageScanning expands the scope of `LocalImageScanningEnabled` to have sensor analyze all images
	// using the local scanner instead of only images from OCP internal registries
	ForceLocalImageScanning = RegisterBooleanSetting("ROX_FORCE_LOCAL_IMAGE_SCANNING", false)

	// EventPipelineQueueSize is used to specify the size of the eventPipeline's queues
	EventPipelineQueueSize = RegisterIntegerSetting("ROX_EVENT_PIPELINE_QUEUE_SIZE", 100)

	// ResyncDisabled disables the resync behavior of the kubernetes listeners in sensor
	ResyncDisabled = RegisterBooleanSetting("ROX_RESYNC_DISABLED", false)
)
