package env

import "time"

var (
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

	// NodeInventoryContainerEnabled is used to tell compliance whether a connection to the node-inventory container should be attempted
	NodeInventoryContainerEnabled = RegisterBooleanSetting("ROX_CALL_NODE_INVENTORY_ENABLED", true)

	// NodeAnalysisDeadline is a time in which node-inventory component should reply to compliance
	NodeAnalysisDeadline = registerDurationSetting("ROX_NODE_SCANNING_DEADLINE", 30*time.Second)

	// NodeScanningAckDeadlineBase defines a base for calculating time when compliance would resend node-inventory
	// if no ACK from central arrives.
	NodeScanningAckDeadlineBase = registerDurationSetting("ROX_NODE_SCANNING_ACK_DEADLINE_BASE", 30*time.Second)
)
