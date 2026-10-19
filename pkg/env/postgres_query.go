package env

var (
	// PostgresParameterThreshold controls when the search framework switches
	// from IN ($1, $2, ...) to = ANY($1::type[]) for exact-match disjunctions.
	// Values below this threshold use IN for better plan quality; values at or
	// above use ANY to avoid the 65535 parameter limit.
	PostgresParameterThreshold = RegisterIntegerSetting("ROX_POSTGRES_PARAMETER_THRESHOLD", 100)

	// PostgresDefaultCursorBatchSize sets the number of rows fetched per FETCH in server-side
	// cursor queries. Lower values reduce memory usage; higher values reduce round-trips.
	PostgresDefaultCursorBatchSize = RegisterIntegerSetting("ROX_POSTGRES_CURSOR_BATCH_SIZE", 1000)
)
