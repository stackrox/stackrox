package env

import "time"

var (
	// NodeScanningInterval is the base value of the interval duration between node scans.
	NodeScanningInterval = registerDurationSetting("ROX_NODE_SCANNING_INTERVAL", 4*time.Hour)

	// NodeScanningIntervalDeviation is the duration node scans will deviate from the
	// base interval time. The value is capped by the base interval.
	NodeScanningIntervalDeviation = registerDurationSetting("ROX_NODE_SCANNING_INTERVAL_DEVIATION", 24*time.Minute)

	// NodeScanningMaxInitialWait is the maximum wait time before the first node
	// scan, which is randomly generated. Set zero to disable the initial node
	// scanning wait time.
	NodeScanningMaxInitialWait = registerDurationSetting("ROX_NODE_SCANNING_MAX_INITIAL_WAIT", 5*time.Minute)

	// NodeInventoryCacheDuration defines the time after which a cached inventory is considered outdated. Defaults to 90% of rescan interval.
	NodeInventoryCacheDuration = registerDurationSetting("ROX_NODE_INVENTORY_CACHE_TIME", time.Duration(NodeRescanInterval.DurationSetting().Nanoseconds()-NodeRescanInterval.DurationSetting().Nanoseconds()/10))

	// NodeInventoryInitialBackoff defines the initial time in seconds a Node Inventory will be delayed if a backoff file is found
	NodeInventoryInitialBackoff = registerDurationSetting("ROX_NODE_INVENTORY_INITIAL_BACKOFF", 30*time.Second)

	// NodeInventoryBackoffIncrement sets the seconds that are added on each interrupted run
	NodeInventoryBackoffIncrement = registerDurationSetting("ROX_NODE_INVENTORY_BACKOFF_INCREMENT", 5*time.Second)

	// NodeInventoryMaxBackoff is the upper boundary of backoff. Defaults to 5m in seconds, being 50% of Kubernetes restart policy stability timer.
	NodeInventoryMaxBackoff = registerDurationSetting("ROX_NODE_INVENTORY_MAX_BACKOFF", 300*time.Second)
)
