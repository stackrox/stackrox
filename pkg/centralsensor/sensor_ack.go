package centralsensor

// Reason constants for SensorACK messages (used with both ACK and NACK actions).
// Shared between Central (sender) and Sensor (receiver) for consistent handling.
const (
	SensorACKReasonRateLimited       = "central rate limit exceeded"
	SensorACKReasonEnrichmentFailed  = "enrichment failed"
	SensorACKReasonStorageFailed     = "storage failed"
	SensorACKReasonMissingScanData   = "missing scanner index data"
	SensorACKReasonMissingClusterID  = "missing cluster ID"
	SensorACKReasonUnsupportedAction = "unsupported action"
	SensorACKReasonFeatureDisabled   = "feature disabled on central"
)
