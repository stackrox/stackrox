package env

import "time"

var (
	// BaseImageWatcherEnabled controls whether the base image watcher is enabled.
	BaseImageWatcherEnabled = RegisterBooleanSetting("ROX_BASE_IMAGE_WATCHER_ENABLED", false)

	// BaseImageWatcherPollInterval controls how often the base image watcher polls for new tags.
	BaseImageWatcherPollInterval = registerDurationSetting("ROX_BASE_IMAGE_WATCHER_POLL_INTERVAL", 4*time.Hour)

	// BaseImageWatcherMaxConcurrentRepositories controls the maximum number of repositories
	// processed concurrently during a poll cycle.
	BaseImageWatcherMaxConcurrentRepositories = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_MAX_CONCURRENT_REPOSITORIES", 10)
)
