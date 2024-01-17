package env

import "time"

var (
	PruneInterval = registerDurationSetting("ROX_PRUNE_INTERVAL", 1*time.Hour)
	OrphanWindow  = registerDurationSetting("ROX_ORPHAN_WINDOW", 30*time.Minute)
	ClusterGCFreq = registerDurationSetting("ROX_CLUSTER_GC_FREQ", 24*time.Hour)
)
