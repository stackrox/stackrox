package env

import "time"

var (
	// PostgresQueryTracer toggles whether to trace Postgres queries and their timing
	PostgresQueryTracer = RegisterBooleanSetting("ROX_POSTGRES_QUERY_TRACER", false)

	// PostgresQueryTracerGraphQLThreshold sets a threshold for how long a GraphQL query must take to be logged
	PostgresQueryTracerGraphQLThreshold = registerDurationSetting("ROX_POSTGRES_QUERY_TRACER_GRAPHQL_THRESHOLD", 1*time.Second)

	// PostgresQueryTracerQueryThreshold sets a threshold for how long an individual Postgres query must take to be logged
	PostgresQueryTracerQueryThreshold = registerDurationSetting("ROX_POSTGRES_QUERY_TRACER_QUERY_THRESHOLD", 200*time.Millisecond)

	// PostgresQueryLogger toggles whether to log Postgres queries for debugging
	PostgresQueryLogger = RegisterBooleanSetting("ROX_POSTGRES_QUERY_LOGGER", false)

	// PostgresKeepTestDB does not destroy the test databases for debugging
	PostgresKeepTestDB = RegisterBooleanSetting("ROX_POSTGRES_TEST_KEEP_DB", false)
)
