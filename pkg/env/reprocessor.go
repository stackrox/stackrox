package env

var (
	// ReprocessorSemaphoreLimit is the maximum number of reprocessed images.
	ReprocessorSemaphoreLimit = RegisterIntegerSetting("ROX_REPROCESSOR_SEMAPHORE_LIMIT", 10)
)
