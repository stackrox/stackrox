package env

var (
	// AlertRenotifDebounceDuration determines the minimum duration that must pass
	// between an alert being resolved, and a new alert being generated for the same deployment-policy pair,
	// such that notifications are sent for the new alert.
	// If it is set to 0 (the default), notifications are always sent, and there is no debouncing.
	AlertRenotifDebounceDuration = registerDurationSetting("ROX_ALERT_RENOTIF_DEBOUNCE_DURATION", 0, WithDurationZeroAllowed())
)
