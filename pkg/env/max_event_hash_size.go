package env

var (
	// MaxEventHashSize sets the max number of event hashes per cluster
	MaxEventHashSize = RegisterIntegerSetting("ROX_MAX_EVENT_HASH_SIZE", 1000000)
)
