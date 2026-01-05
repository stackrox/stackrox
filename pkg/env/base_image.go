package env

import "time"

var (
	// BaseImageWatcherEnabled controls whether the base image watcher is enabled.
	// This setting is only consulted when the ROX_BASE_IMAGE_DETECTION feature flag is true.
	BaseImageWatcherEnabled = RegisterBooleanSetting("ROX_BASE_IMAGE_WATCHER_ENABLED", true)

	// BaseImageWatcherPollInterval controls how often the base image watcher polls for new tags.
	BaseImageWatcherPollInterval = registerDurationSetting("ROX_BASE_IMAGE_WATCHER_POLL_INTERVAL", 4*time.Hour)

	// BaseImageWatcherMaxConcurrentRepositories controls the maximum number of repositories
	// processed concurrently during a poll cycle.
	BaseImageWatcherMaxConcurrentRepositories = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_MAX_CONCURRENT_REPOSITORIES", 10)

	// BaseImageWatcherRegistryRateLimit controls the maximum requests per second to each
	// registry integration. The default of 5 req/s balances performance with rate limit safety.
	// Lower to 1-2 for unauthenticated Docker Hub or aggressive rate-limited registries.
	BaseImageWatcherRegistryRateLimit = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_REGISTRY_RATE_LIMIT", 5)
)
