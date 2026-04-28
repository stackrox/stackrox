package env

var (
	// ListAlertUseLegacyQuery reverts SearchListAlerts and WalkAll to blob deserialization.
	ListAlertUseLegacyQuery = RegisterBooleanSetting("ROX_LIST_ALERT_LEGACY_QUERY", false)
)
