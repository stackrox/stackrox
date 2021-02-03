package common

var (
	// TempStoragePath is the path to the directory in which the admission control service
	// can temporarily store data (persisted across restarts, but not across pod deletions).
	TempStoragePath = `/var/lib/stackrox/admission-control/`
)
