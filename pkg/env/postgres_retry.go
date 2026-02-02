package env

import "time"

var (
	// PostgresQueryRetryInterval is the interval between retry attempts for transient PostgreSQL errors
	PostgresQueryRetryInterval = registerDurationSetting("ROX_POSTGRES_QUERY_RETRY_INTERVAL", 5*time.Second)

	// PostgresQueryRetryTimeout is the maximum duration to retry transient PostgreSQL errors
	PostgresQueryRetryTimeout = registerDurationSetting("ROX_POSTGRES_QUERY_RETRY_TIMEOUT", 5*time.Minute)

	// PostgresDisableQueryRetries disables retry logic for transient PostgreSQL errors (fail fast after single attempt)
	PostgresDisableQueryRetries = RegisterBooleanSetting("ROX_POSTGRES_DISABLE_QUERY_RETRIES", false)
)
