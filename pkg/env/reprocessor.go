package env

var (
	// ReprocessorSemaphoreLimit is the maximum number of images being reprocessed in parallel.
	ReprocessorSemaphoreLimit = RegisterIntegerSetting("ROX_REPROCESSOR_SEMAPHORE_LIMIT", 10)
)
