package env

var (
	// NodeIndexEnabled defines whether Compliance will actually run indexing code
	NodeIndexEnabled = RegisterBooleanSetting("ROX_NODE_INDEX_ENABLED", false)

	// NodeIndexHostPath sets the path where the R/O host node filesystem is mounted to the container
	// that should be scanned by Scanners NodeIndexer
	NodeIndexHostPath = RegisterSetting("ROX_NODE_INDEX_HOST_PATH", WithDefault("/host"))

	// NodeIndexContainerAPI Defines the API endpoint for the RepositoryScanner to reach out to
	NodeIndexContainerAPI = RegisterSetting("ROX_NODE_INDEX_CONTAINER_API", WithDefault("https://catalog.redhat.com/api/containers/"))

	// NodeIndexMappingURL Defines the endpoint for the RepositoryScanner to download mapping information from
	NodeIndexMappingURL = RegisterSetting("ROX_NODE_INDEX_MAPPING_URL", WithDefault("https://sensor.stackrox.svc/scanner/definitions?file=repo2cpe"))
)
