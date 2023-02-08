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

	// NodeInventoryCacheDuration defines the time after which a cached inventory is considered outdated
	NodeInventoryCacheDuration = registerDurationSetting("ROX_NODE_INVENTORY_CACHE_TIME", 10*time.Minute)

	// NodeInventoryInitialBackoff defines the initial time in seconds a Node Inventory will be delayed if a backoff file is found
	NodeInventoryInitialBackoff = RegisterIntegerSetting("ROX_NODE_INVENTORY_INITIAL_BACKOFF", 30)

	// NodeInventoryBackoffIncrement sets the seconds that are added on each interrupted run
	NodeInventoryBackoffIncrement = RegisterIntegerSetting("ROX_NODE_INVENTORY_BACKOFF_INCREMENT", 5)

	// NodeInventoryMaxBackoff is the upper boundary of backoff. Defaults to 3h50m in seconds
	NodeInventoryMaxBackoff = RegisterIntegerSetting("ROX_NODE_INVENTORY_MAX_BACKOFF", 13800)
)
