package env

var (
	// RedisEndpoint configures the Redis server address for shared caching in HA deployments.
	RedisEndpoint = RegisterSetting("ROX_REDIS_ENDPOINT", WithDefault("redis.stackrox.svc:6379"))
)
