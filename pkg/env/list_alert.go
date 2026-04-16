package env

var (
	// ListAlertUseLegacyQuery controls whether SearchListAlerts and WalkAll
	// use the legacy blob deserialization path instead of the optimized column
	// projection path. Set to true to revert to the old behavior if the
	// projection path causes issues.
	ListAlertUseLegacyQuery = RegisterBooleanSetting("ROX_LIST_ALERT_LEGACY_QUERY", false)
)
