package env

import "time"

var (
	// ComplianceScanTimeout defines the timeout for compliance scan. If the scan hasn't finished by then, it will be aborted.
	// Compliance operator default is 30 mins.
	ComplianceScanTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_TIMEOUT", 15*time.Minute)
	// ComplianceScanRetries defines is the maximum number of times the scan will be retried if it times out.
	// Compliance operator default is 3.
	ComplianceScanRetries = RegisterIntegerSetting("ROX_COMPLIANCE_SCAN_RETRY_COUNT", 2)
	// ComplianceStrictNodeScan defines if scans can proceed if the scan should fail if any node cannot be scanned
	ComplianceStrictNodeScan = RegisterBooleanSetting("ROX_COMPLIANCE_STRICT_NODE_SCAN", true)

	// ComplianceScanWatcherTimeout defines the timeout for a compliance scan watcher.
	// If the scan results have not been received by then, it will be aborted.
	// The default is 40 mins.
	ComplianceScanWatcherTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT", 40*time.Minute)

	// ComplianceScanScheduleWatcherTimeout defines the timeout for a compliance scan schedule watcher.
	// If the scan results of all scans associated with the schedule have not been received by then, it will be aborted.
	// The default is 45 mins.
	ComplianceScanScheduleWatcherTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_SCHEDULE_WATCHER_TIMEOUT", 45*time.Minute)

	// ComplianceReportSettleTimeout bounds the settle phase of the settle-audit-snapshot
	// report primitive (ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER): how long a report waits for
	// the scans of the scan configuration that are currently running before it proceeds and
	// labels them as unfinished. The default is 10 mins.
	ComplianceReportSettleTimeout = registerDurationSetting("ROX_COMPLIANCE_REPORT_SETTLE_TIMEOUT", 10*time.Minute)

	// ComplianceReportSettlePollInterval is how often the settle phase re-checks the scan
	// datastore for scans that reached a terminal state. The default is 15 secs.
	ComplianceReportSettlePollInterval = registerDurationSetting("ROX_COMPLIANCE_REPORT_SETTLE_POLL_INTERVAL", 15*time.Second)

	// ComplianceReportScheduleGrace absorbs Compliance Operator and sensor scheduling delays
	// (and the UTC approximation of the cluster-local cron) when the report audit decides
	// whether a scan started within the expected schedule window. The default is 1 hour.
	ComplianceReportScheduleGrace = registerDurationSetting("ROX_COMPLIANCE_REPORT_SCHEDULE_GRACE", 1*time.Hour)

	// ComplianceReportSchedulerResyncInterval is how often the report scheduler
	// (ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER) reconciles its per-configuration schedulers
	// against the scan configurations in the database. It is a safety net for missed
	// create/delete notifications (the primary triggers are the scan-config pipeline
	// callbacks and the schedulers' own due-time wake-ups). The default is 1 minute.
	ComplianceReportSchedulerResyncInterval = registerDurationSetting("ROX_COMPLIANCE_REPORT_SCHEDULER_RESYNC_INTERVAL", 1*time.Minute)

	// ComplianceReportScheduleDelay is the offset between a scan configuration's schedule
	// firing and the scheduled report for that fire: the report runs at fire + delay, giving
	// the scans time to finish before the report settles and snapshots them. The settle
	// phase and the audit absorb the imprecision. It must stay well below the scan
	// interval (the default of 1 hour vs. daily-or-slower scans leaves ample margin).
	// A separate user-facing report cron is the cleaner long-term option (see the
	// "compliance reporting on partial scan cycles" proposal, open product decisions).
	ComplianceReportScheduleDelay = registerDurationSetting("ROX_COMPLIANCE_REPORT_SCHEDULE_DELAY", 1*time.Hour)

	// ComplianceMaxNumberOfErrorsInReport defines the max number of errors that a report will store. This is done to avoid overwhelming the UI with many errors.
	// The default is 4
	ComplianceMaxNumberOfErrorsInReport = RegisterIntegerSetting("ROX_COMPLIANCE_MAX_NUMBER_OF_ERRORS_IN_REPORT", 4)

	// ComplianceMinimalSupportedVersion specifies the minimum version of the compliance operator that is supported by StackRox.
	// This value can be customized via the ROX_COMPLIANCE_MINIMAL_SUPPORTED_OPERATOR_VERSION environment variable.
	// If the environment variable is unset, contains an invalid version, or is lower than the default value, the default value "v1.6.0" will be used.
	ComplianceMinimalSupportedVersion = RegisterVersionSetting("ROX_COMPLIANCE_MINIMAL_SUPPORTED_OPERATOR_VERSION", "v1.6.0", "v1.6.0")
	// ComplianceScansRunningInParallelMetricObservationPeriod defines an observation window for the compliance operator metrics in Central.
	// For example, a metric output like this:
	// rox_central_complianceoperator_num_scans_running_in_parallel_bucket{le="0"} 0
	// rox_central_complianceoperator_num_scans_running_in_parallel_bucket{le="1"} 6
	// rox_central_complianceoperator_num_scans_running_in_parallel_bucket{le="2"} 16
	// rox_central_complianceoperator_num_scans_running_in_parallel_bucket{le="3"} 20
	// rox_central_complianceoperator_num_scans_running_in_parallel_bucket{le="4"} 36
	// with setting of 1 hour, should be interpreted as follows:
	// - The line with `le="0"` should be always empty, as we never have a negative number of scans running
	// - There were six 1-hour-long time-windows with not a single scan was running
	// - There were ten (16-6) 1-hour-long time-windows with a single scan was running
	// - There were four (20-16) 1-hour-long time-windows with two scans were running in parallel
	// - There were fourteen (34-20) 1-hour-long time-windows with three scans were running in parallel
	ComplianceScansRunningInParallelMetricObservationPeriod = registerDurationSetting("ROX_COMPLIANCE_METRIC_OBSERVATION_PERIOD", 15*time.Minute)
)
