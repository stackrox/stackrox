package env

import "time"

var (
	// RHCOSNodeScanning enables phase 1 functions of "Full host level vulnerability scanning for RHCOS nodes" (ROX-10818)
	RHCOSNodeScanning = RegisterBooleanSetting("ROX_RHCOS_NODE_SCANNING", true)

	// NodeScanningEndpoint is used to provide Compliance with the Node Scanner that is used to carry out Node Scans
	NodeScanningEndpoint = RegisterSetting("ROX_NODE_SCANNING_ENDPOINT", WithDefault("127.0.0.1:8444"))

	// NodeScanningInterval is the base value of the interval duration between node scans.
	NodeScanningInterval = registerDurationSetting("ROX_NODE_SCANNING_INTERVAL", 4*time.Hour)

	// NodeScanningIntervalDeviation is the duration node scans will deviate from the
	// base interval time. The value is capped by the base interval.
	NodeScanningIntervalDeviation = registerDurationSetting("ROX_NODE_SCANNING_INTERVAL_DEVIATION", 24*time.Minute)

	// NodeScanningMaxInitialWait is the maximum wait time before the first node
	// scan, which is randomly generated. Set zero to disable the initial node
	// scanning wait time.
	NodeScanningMaxInitialWait = registerDurationSetting("ROX_NODE_SCANNING_MAX_INITIAL_WAIT", 5*time.Minute)

	// NodeScanningCacheDuration defines the time after which a cached inventory is considered outdated. Defaults to 90% of NodeScanningInterval.
	NodeScanningCacheDuration = registerDurationSetting("ROX_NODE_SCANNING_CACHE_TIME", time.Duration(NodeScanningInterval.DurationSetting().Nanoseconds()-NodeScanningInterval.DurationSetting().Nanoseconds()/10))

	// NodeScanningInitialBackoff defines the initial time in seconds a Node scan will be delayed if a backoff file is found
	NodeScanningInitialBackoff = registerDurationSetting("ROX_NODE_SCANNING_INITIAL_BACKOFF", 30*time.Second)

	// NodeScanningMaxBackoff is the upper boundary of backoff. Defaults to 5m in seconds, being 50% of Kubernetes restart policy stability timer.
	NodeScanningMaxBackoff = registerDurationSetting("ROX_NODE_SCANNING_MAX_BACKOFF", 300*time.Second)

	// NodeInventoryContainerEnabled is used to tell compliance whether a connection to the node-inventory container should be attempted
	NodeInventoryContainerEnabled = RegisterBooleanSetting("ROX_CALL_NODE_INVENTORY_ENABLED", true)
)
