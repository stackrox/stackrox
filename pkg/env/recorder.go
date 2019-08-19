package env

import "time"

var (
	// RecorderTime will set the duration for which to record the events from Sensor. 0 is the default which means no recording
	RecorderTime = registerDurationSetting("ROX_RECORDER_DURATION", 0*time.Minute)
)
