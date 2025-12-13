package env

var (
	// CentralSensorWorkerQueueSize controls the per-event-type worker queue parallelism in Central.
	// It defaults to 16, matching historical behavior.
	CentralSensorWorkerQueueSize = RegisterIntegerSetting("ROX_CENTRAL_SENSOR_WORKER_QUEUE_SIZE", 16).WithMinimum(1)

	// CentralSensorWorkerQueueDepth caps the per-worker queue depth to avoid unbounded memory.
	// Default 25 matches the previous hardcoded limit; must be >=1 to avoid disabling protection.
	CentralSensorWorkerQueueDepth = RegisterIntegerSetting("ROX_CENTRAL_SENSOR_WORKER_QUEUE_DEPTH", 25).WithMinimum(1)
)
