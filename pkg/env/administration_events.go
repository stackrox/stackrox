package env

import "time"

var (
	// AdministrationEventFlushInterval is the interval in which administration events will be flushed from the buffer
	// and written to the database.
	AdministrationEventFlushInterval = registerDurationSetting("ROX_ADMINISTRATION_EVENTS_FLUSH_INTERVAL",
		time.Minute)
)
