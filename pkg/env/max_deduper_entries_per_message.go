package env

var (
	// MaxDeduperEntriesPerMessage sets the max number of deduper entries per message to be sent during the sync
	MaxDeduperEntriesPerMessage = RegisterIntegerSetting("ROX_MAX_DEDUPER_ENTRIES_PER_MESSAGE", 200)
)
