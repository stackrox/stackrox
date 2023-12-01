package env

import "time"

// These environment variables are used in the deployment file
// Please check the files before deleting
var (
	// CentralEndpoint is used to provide Central's reachable endpoint to a sensor.
	CentralEndpoint = RegisterSetting("ROX_CENTRAL_ENDPOINT", WithDefault("central.stackrox.svc:443"),
		StripAnyPrefix("https://", "http://"))

	// AdvertisedEndpoint is used to provide the Sensor with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = RegisterSetting("ROX_ADVERTISED_ENDPOINT", WithDefault("sensor.stackrox.svc:443"))

	// SensorEndpoint is used to communicate the sensor endpoint to other services in the same cluster.
	SensorEndpoint = RegisterSetting("ROX_SENSOR_ENDPOINT", WithDefault("sensor.stackrox.svc:443"))

	// ScannerSlimGRPCEndpoint is used to communicate the scanner endpoint to other services in the same cluster.
	// This is typically used for Sensor to communicate with a local Scanner-slim's gRPC server.
	ScannerSlimGRPCEndpoint = RegisterSetting("ROX_SCANNER_GRPC_ENDPOINT", WithDefault("scanner.stackrox.svc:8443"))

	// ScannerV4GRPCEndpoint is used to communicate with the Scanner V4 endpoint in the same cluster.
	ScannerV4GRPCEndpoint = RegisterSetting("ROX_SCANNER_V4_GRPC_ENDPOINT", WithDefault("scanner-v4.stackrox.svc:8443"))

	// ScannerV4Enabled Enables Scanner V4 as the default StackRox image scanner in Sensor.
	ScannerV4Enabled = RegisterBooleanSetting("ROX_SCANNER_V4_ENABLED", false)

	// LocalImageScanningEnabled is used to specify if Sensor should attempt to scan images via a local Scanner.
	LocalImageScanningEnabled = RegisterBooleanSetting("ROX_LOCAL_IMAGE_SCANNING_ENABLED", false)

	// EventPipelineQueueSize is used to specify the size of the eventPipeline's queues
	EventPipelineQueueSize = RegisterIntegerSetting("ROX_EVENT_PIPELINE_QUEUE_SIZE", 1000)

	// ResyncDisabled disables the resync behavior of the kubernetes listeners in sensor
	ResyncDisabled = RegisterBooleanSetting("ROX_RESYNC_DISABLED", true)

	// ConnectionRetryInitialInterval defines how long it takes for sensor to retry gRPC connection when it first disconnects.
	ConnectionRetryInitialInterval = registerDurationSetting("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", 10*time.Second)

	// ConnectionRetryMaxInterval defines the maximum interval between retries after the gRPC connection disconnects.
	ConnectionRetryMaxInterval = registerDurationSetting("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", 5*time.Minute)

	// DelegatedScanningDisabled disables the capabilities associated with delegated image scanning.
	// This is meant to be a 'kill switch' that allows for local scanning to continue (ie: for OCP internal repos)
	// in the event the delegated scanning capabilities are causing unforeseen issues.
	DelegatedScanningDisabled = RegisterBooleanSetting("ROX_DELEGATED_SCANNING_DISABLED", false)

	// RegistryTLSCheckTTL will set the duration for which registry TLS checks will be cached.
	RegistryTLSCheckTTL = registerDurationSetting("ROX_SENSOR_REGISTRY_TLS_CHECK_CACHE_TTL", 15*time.Minute)

	// DeduperStateSyncTimeout defines the maximum time Sensor will wait for the expected deduper state coming from Central
	DeduperStateSyncTimeout = registerDurationSetting("ROX_DEDUPER_STATE_TIMEOUT", 30*time.Second)
)
