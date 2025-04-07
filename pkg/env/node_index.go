package env

import "time"

var (
	// NodeIndexHostPath sets the path where the R/O host node filesystem is mounted to the container.
	// that should be scanned by Scanners NodeIndexer
	NodeIndexHostPath = RegisterSetting("ROX_NODE_INDEX_HOST_PATH", WithDefault("/host"))

	// NodeIndexMappingURL defines the endpoint for the RepositoryScanner to download mapping information from.
	// If left empty, the URL will be computed based on Sensor's ROX_ADVERTISED_ENDPOINT.
	// The default "https://sensor.stackrox.svc/scanner/definitions?file=repo2cpe" is not set here to not hardcode the namespace of Sensor.
	NodeIndexMappingURL = RegisterSetting("ROX_NODE_INDEX_MAPPING_URL", AllowEmpty())

	// NodeIndexCacheDuration defines the time a cached node index will be considered fresh and served from file.
	// Defaults to 75% of the default rescan interval.
	NodeIndexCacheDuration = registerDurationSetting("ROX_NODE_INDEX_CACHE_DURATION", 3*time.Hour)

	// NodeIndexCachePath defines the path to the file where the node index wrap cache will be written to.
	// This path is expected to be writable inside the Compliance container.
	NodeIndexCachePath = RegisterSetting("ROX_NODE_INDEX_CACHE_PATH", WithDefault("/tmp/node-index"))
)
