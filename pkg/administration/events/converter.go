package events

// LogConverter converts a log entry to an events.AdministrationEvent.
type LogConverter interface {
	Convert(msg string, level string, module string, context ...interface{}) *AdministrationEvent
}
