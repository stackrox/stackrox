package env

var (
	// AdmissionControlImageCacheSizeMB controls the maximum size (in megabytes) of
	// the in-process image cache used by the admission controller for policy evaluation.
	// The Helm chart sets this based on whether enforcement is enabled (500 MB) or
	// disabled (200 MB). Operators can override via the environment variable.
	AdmissionControlImageCacheMaxSizeMB = RegisterIntegerSetting("ROX_ADMISSION_CONTROL_IMAGE_CACHE_MAX_SIZE", 200)
)
