package env

var (
	CentralWorkerEnabled            = RegisterBooleanSetting("ROX_CENTRAL_WORKER_ENABLED", false)
	CentralWorkerResyncIntervalMins = RegisterIntegerSetting("ROX_WORKER_RESYNC_INTERVAL_MINUTES", 5).WithMinimum(1)
)
