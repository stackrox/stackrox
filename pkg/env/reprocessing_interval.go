package env

import "time"

var (
	// RiskReprocessInterval will set the duration for which to debounce risk reprocessing
	RiskReprocessInterval = registerDurationSetting("ROX_RISK_REPROCESSING_INTERVAL", 10*time.Minute)
	// ReprocessInterval will set the duration for which to reprocess all deployments and get new scans
	ReprocessInterval = registerDurationSetting("ROX_REPROCESSING_INTERVAL", 4*time.Hour)
	// VulnDeferralTimedReObserveInterval will set the duration for when to check to see if timed vuln deferrals need to be checked for expiry.
	VulnDeferralTimedReObserveInterval = registerDurationSetting("ROX_VULN_TIMED_DEFERRAL_REOBSERVE_INTERVAL", 1*time.Hour)
	// VulnDeferralFixableReObserveInterval will set the duration for when to check to see if "when fixable" vuln deferrals need to be checked for expiry.
	VulnDeferralFixableReObserveInterval = registerDurationSetting("ROX_VULN_FIXABLE_DEFERRAL_REOBSERVE_INTERVAL", 4*time.Hour)
	// OrchestratorVulnScanInterval specifies the frequency at which Central should scan for new orchestrator-level vulnerabilities.
	OrchestratorVulnScanInterval = registerDurationSetting("ROX_ORCHESTRATOR_VULN_SCAN_INTERVAL", 2*time.Hour)
	// ReprocessInjectMessageTimeout specifies the duration to wait when sending a message to sensor during reprocessing. If this duration
	// is exceeded subsequent messages targeting this particular sensor will be skipped until the next reprocessing cycle.
	// Setting the duration to zero will disable the timeout.
	ReprocessInjectMessageTimeout = registerDurationSetting("ROX_REPROCESSING_INJECT_MESSAGE_TIMEOUT", 1*time.Minute, WithDurationZeroAllowed())
	// ReprocessDeploymentsMsgDelay specifies the delay to wait between sending "ReprocessDeployments"
	// messages to Sensors at the end of Central image reprocessing. When set to 0, messages are sent as fast
	// as possible
	ReprocessDeploymentsMsgDelay = registerDurationSetting("ROX_REPROCESS_DEPLOYMENTS_MSG_DELAY", 0, WithDurationZeroAllowed())
	// DeploymentRiskMaxConcurrency limits how many deployments can have their risk reprocessed
	// concurrently across all clusters. Each reprocessing operation makes multiple DB calls
	// (risk lookups, baseline evaluations, process indicator queries, upserts). Without a limit,
	// 17 worker goroutines per cluster can overwhelm the connection pool (default 90) when
	// multiple clusters trigger reprocessing simultaneously (e.g., sensor reconnect, periodic enrichment).
	// Default of 15 provides headroom: ~15 concurrent reprocessing operations use ~15 DB connections,
	// leaving the remaining ~75 connections available for all other Central operations.
	DeploymentRiskMaxConcurrency = RegisterIntegerSetting("ROX_DEPLOYMENT_RISK_MAX_CONCURRENCY", 15).
				WithMinimum(1).
				WithMaximum(30)
	// DeploymentRiskSemaphoreWaitTime is the maximum time a worker will wait to acquire
	// the risk reprocessing semaphore. If exceeded, the operation is dropped and will be
	// retried on the next reprocessing cycle. Setting to zero disables the timeout (workers
	// block indefinitely until a slot is available or the sensor disconnects).
	DeploymentRiskSemaphoreWaitTime = registerDurationSetting("ROX_DEPLOYMENT_RISK_SEMAPHORE_WAIT_TIME", 2*time.Minute, WithDurationZeroAllowed())
)
