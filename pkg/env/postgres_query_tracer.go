package env

var (
	// PostgresQueryTracer toggles whether to trace Postgres queries and their timing
	PostgresQueryTracer = RegisterBooleanSetting("ROX_POSTGRES_QUERY_TRACER", false)
)
