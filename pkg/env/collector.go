package env

var (
	// CollectorVersion is the version tag to be used for the collector image
	// It is used by the collector team to manually override the collector version in the COLLECTOR_VERSION file
	CollectorVersion = RegisterSetting("ROX_COLLECTOR_VERSION")
)
