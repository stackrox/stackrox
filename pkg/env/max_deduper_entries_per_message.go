package env

const (
	defaultMaxDeduperEntriesPerMessage = 200000
)

var (
	// MaxDeduperEntriesPerMessage sets the max number of deduper entries per message to be sent during the sync
	MaxDeduperEntriesPerMessage = RegisterIntegerSetting("ROX_MAX_DEDUPER_ENTRIES_PER_MESSAGE", defaultMaxDeduperEntriesPerMessage)
	TmpCrashGrpcAfterMessages   = RegisterIntegerSetting("ROX_TMP_NUM_MSG_CRASH_GRPC", 0)
)

// GetMaxDeduperEntriesPerMessage returns a sanitized MaxDeduperEntriesPerMessage
func GetMaxDeduperEntriesPerMessage() int32 {
	v := MaxDeduperEntriesPerMessage.IntegerSetting()
	if v < 1 {
		return defaultMaxDeduperEntriesPerMessage
	}
	return int32(v)
}
