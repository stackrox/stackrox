package env

var (
	// NotifyEveryRuntimeEvent toggles whether every runtime event causes an event notification
	NotifyEveryRuntimeEvent = RegisterBooleanSetting("NOTIFY_EVERY_RUNTIME_EVENT", true)
)

// NotifyOnEveryRuntimeEvent returns true if we should notify on every runtime event
func NotifyOnEveryRuntimeEvent() bool {
	return NotifyEveryRuntimeEvent.BooleanSetting()
}
