package env

import "time"

var (
	// NodeIndexEnabled defines whether Compliance will actually run indexing code.
	NodeIndexEnabled = RegisterBooleanSetting("ROX_NODE_INDEX_ENABLED", false)

	// NodeIndexHostPath sets the path where the R/O host node filesystem is mounted to the container.
	// that should be scanned by Scanners NodeIndexer
	NodeIndexHostPath = RegisterSetting("ROX_NODE_INDEX_HOST_PATH", WithDefault("/host"))

	// NodeIndexContainerAPI defines the API endpoint for the RepositoryScanner to reach out to.
	NodeIndexContainerAPI = RegisterSetting("ROX_NODE_INDEX_CONTAINER_API", WithDefault("https://catalog.redhat.com/api/containers/"))

	// NodeIndexMappingURL defines the endpoint for the RepositoryScanner to download mapping information from.
	NodeIndexMappingURL = RegisterSetting("ROX_NODE_INDEX_MAPPING_URL", WithDefault("https://sensor.stackrox.svc/scanner/definitions?file=repo2cpe"))

	// NodeIndexCacheDuration defines the time a cached node index will be considered fresh and served from file.
	// Defaults to 75% of the default rescan interval.
	NodeIndexCacheDuration = registerDurationSetting("ROX_NODE_INDEX_CACHE_DURATION", 3*time.Hour)

	// NodeIndexCachePath defines the path to the file where the node index wrap cache will be written to.
	// This path is expected to be writable inside the Compliance container.
	NodeIndexCachePath = RegisterSetting("ROX_NODE_INDEX_CACHE_PATH", WithDefault("/tmp/node-index"))
)
