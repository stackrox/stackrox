package env

import "time"

var (
	// HashFlushInterval sets the frequency of flushing the received hashes to the database
	HashFlushInterval = registerDurationSetting("ROX_HASH_FLUSH_INTERVAL", 1*time.Minute)
)
