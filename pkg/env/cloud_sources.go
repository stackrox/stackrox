package env

import "time"

var (
	// DiscoveredClustersRetentionTime is the retention time for discovered clusters.
	DiscoveredClustersRetentionTime = registerDurationSetting("ROX_DISCOVERED_CLUSTERS_RETENTION_TIME",
		24*time.Hour)
)
