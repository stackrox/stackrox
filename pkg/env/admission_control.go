package env

import "time"

var (
	// AdmissionControlImageCacheMaxSizeMB AdmissionControlImageCacheSizeMB controls the maximum size (in megabytes) of
	// the in-process image cache used by the admission controller for policy evaluation.
	// The Helm chart sets this based on whether enforcement is enabled (500 MB) or
	// disabled (200 MB). Operators can override via the environment variable.
	AdmissionControlImageCacheMaxSizeMB = RegisterIntegerSetting("ROX_ADMISSION_CONTROL_IMAGE_CACHE_MAX_SIZE", 200)

	// AdmissionControlImageNameCacheEnabled controls whether the admission controller
	// caches the mapping from image names (e.g. "nginx:1.25") to their resolved cache
	// keys. When enabled, tag-only images can hit the image scan cache across requests without
	// requiring to be re-fetched. Disable for workflows where image tags are frequently repointed
	// to new digests without being re-tagged (mutable tags).
	AdmissionControlImageNameCacheEnabled = RegisterBooleanSetting("ROX_ADMISSION_CONTROL_IMAGE_NAME_CACHE_ENABLED", true)

	// AdmissionControlImageCacheTTL controls how long enriched image entries remain valid
	// in the admission controller's in-process cache before being evicted and re-fetched.
	// Aligned with the Central reprocessing interval by default.
	AdmissionControlImageCacheTTL = registerDurationSetting("ROX_ADMISSION_CONTROL_IMAGE_CACHE_TTL", 4*time.Hour)
)
