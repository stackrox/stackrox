package env

import "time"

var (
	// ReprocessInterval will set the duration for which to reprocess all deployments and get new scans
	ReprocessInterval = registerDurationSetting("ROX_REPROCESSING_INTERVAL", 4*time.Hour)
)
