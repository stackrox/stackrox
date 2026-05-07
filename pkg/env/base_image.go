package env

import "time"

var (
	// BaseImageWatcherEnabled controls whether the base image watcher is enabled.
	// This setting is only consulted when the ROX_BASE_IMAGE_DETECTION feature flag is true.
	BaseImageWatcherEnabled = RegisterBooleanSetting("ROX_BASE_IMAGE_WATCHER_ENABLED", true)

	// BaseImageWatcherPollInterval controls the per-repository polling cadence.
	// A repository is due for scanning when last_polled_at + this interval < now.
	BaseImageWatcherPollInterval = registerDurationSetting("ROX_BASE_IMAGE_WATCHER_POLL_INTERVAL", 4*time.Hour)

	// BaseImageWatcherSchedulerCadence controls how often the scheduler checks for due repositories.
	// Lower values reduce latency for newly created repositories but increase scheduling overhead.
	BaseImageWatcherSchedulerCadence = registerDurationSetting("ROX_BASE_IMAGE_WATCHER_SCHEDULER_CADENCE", 1*time.Minute)

	// BaseImageWatcherMaxConcurrentRepositories controls the maximum number of repositories
	// processed concurrently during a poll cycle.
	BaseImageWatcherMaxConcurrentRepositories = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_MAX_CONCURRENT_REPOSITORIES", 10)

	// BaseImageWatcherRegistryRateLimit controls the maximum requests per second to each
	// registry integration. The default of 5 req/s balances performance with rate limit safety.
	// Lower to 1-2 for unauthenticated Docker Hub or aggressive rate-limited registries.
	BaseImageWatcherRegistryRateLimit = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_REGISTRY_RATE_LIMIT", 5)

	// BaseImageWatcherPerRepoTagLimit controls the maximum number of tags promoted from cache
	// (base_image_tags) to active list (base_images) per repository.
	BaseImageWatcherPerRepoTagLimit = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_PER_REPO_TAG_LIMIT", 100)

	// BaseImageWatcherTagBatchSize controls the number of tag operations (upserts or deletes)
	// to accumulate before flushing to the database. Larger batches reduce database round-trips
	// but increase memory usage. Smaller batches reduce memory pressure but increase DB calls.
	BaseImageWatcherTagBatchSize = RegisterIntegerSetting("ROX_BASE_IMAGE_WATCHER_TAG_BATCH_SIZE", 100)
)
