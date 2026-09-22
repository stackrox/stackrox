package env

import "time"

var (
	// NodeIndexHostPath sets the path where the R/O host node filesystem is mounted to the container.
	// that should be scanned by Scanners NodeIndexer
	NodeIndexHostPath = RegisterSetting("ROX_NODE_INDEX_HOST_PATH", WithDefault("/host"))
	// NodeIndexOSReleasePath sets the path where os-release can be found for RHCOS detection.
	// This may differ from NodeIndexHostPath when the RPM database is mounted separately.
	NodeIndexOSReleasePath = RegisterSetting("ROX_NODE_INDEX_OSRELEASE_PATH", WithDefault("/host"))
	// NodeIndexMappingURL defines the endpoint for the RepositoryScanner to download mapping information from.
	// If left empty, the URL will be computed based on Sensor's ROX_SENSOR_ENDPOINT (see SensorEndpointSetting).
	// The default "https://sensor.stackrox.svc/scanner/definitions?file=repo2cpe" is not set here to not hardcode the namespace of Sensor.
	NodeIndexMappingURL = RegisterSetting("ROX_NODE_INDEX_MAPPING_URL", AllowEmpty())
	// NodeIndexCacheDuration defines the time a cached node index will be considered fresh and served from file.
	// Defaults to 75% of the default rescan interval.
	NodeIndexCacheDuration = registerDurationSetting("ROX_NODE_INDEX_CACHE_DURATION", 3*time.Hour)
	// NodeIndexCachePath defines the path to the file where the node index wrap cache will be written to.
	// This path is expected to be writable inside the Compliance container.
	NodeIndexCachePath = RegisterSetting("ROX_NODE_INDEX_CACHE_PATH", WithDefault("/tmp/node-index"))

	// NodeIndexReportRateLimit is the maximum number of node index reports per second Central
	// accepts across all sensors that send them. Each such sensor gets an equal share (1/N) of
	// this global capacity. The split happens when a new client ID registers in the node index
	// report pipeline.
	// Supports fractional rates (e.g., "0.5" for one request every 2 seconds).
	// Set to "0" to disable rate limiting (unlimited).
	//
	// Default 0.1 is one report every 10 seconds, a conservative starting point for Scanner V4 load.
	NodeIndexReportRateLimit = RegisterFloatSetting("ROX_NODE_INDEX_REPORT_RATE_LIMIT", 0.1)

	// NodeIndexReportBucketCapacity is the token-bucket capacity for node index report rate limiting.
	// This is the maximum number of requests that can be accepted in a burst before rate limiting
	// kicks in. The global capacity is divided equally among connected sensors.
	// Default: 3 tokens.
	NodeIndexReportBucketCapacity = RegisterIntegerSetting("ROX_NODE_INDEX_REPORT_BUCKET_CAPACITY", 3).WithMinimum(1)
)
