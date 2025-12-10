package env

import "time"

var (
	// AdministrationEventFlushInterval is the interval in which administration events will be flushed from the buffer
	// and written to the database.
	AdministrationEventFlushInterval = registerDurationSetting("ROX_ADMINISTRATION_EVENTS_FLUSH_INTERVAL",
		time.Minute)

	// AdministrationEventsAdHocScans when true will cause failed image scans from ad hoc sources, such as roxctl, to
	// generate admin events.
	AdministrationEventsAdHocScans = RegisterBooleanSetting("ROX_ADHOC_SCAN_ADMIN_EVENTS", true)
)
