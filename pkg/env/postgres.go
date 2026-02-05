package env

import "time"

var (
	// PostgresMonitoringInterval specifies how frequently the database
	// statistics will be collected. Every time it happens a set of queries are
	// performed against the database to monitor its internal state: tables
	// size, number of tuples and open connections. In real environment those
	// metrics change slowly, and there is no reason to hit the database every
	// minute or so to get the same numbers over and over. But for testing
	// environment it might be beneficial to increase monitoring resolution.
	PostgresMonitoringInterval = registerDurationSetting("ROX_POSTGRES_DEFAULT_MONITORING_INTERVAL", 1*time.Hour)
)
