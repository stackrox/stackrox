package env

import "time"

var (
	// BaseImageEnable controls whether the base image detection feature is enabled.
	BaseImageEnable = RegisterBooleanSetting("ROX_BASE_IMAGE_ENABLE", false)

	// BaseImagePollInterval controls how often the base image watcher polls for new tags.
	BaseImagePollInterval = registerDurationSetting("ROX_BASE_IMAGE_POLL_INTERVAL", 4*time.Hour)

	// BaseImageMaxConcurrentRepositories controls the maximum number of repositories
	// processed concurrently during a poll cycle.
	BaseImageMaxConcurrentRepositories = RegisterIntegerSetting("ROX_BASE_IMAGE_MAX_CONCURRENT_REPOSITORIES", 10)
)
