package env

import "time"

var (
	// Phone-Home telemetry variables:

	// TelemetryEndpoint is the endpoint to which to send telemetry data.
	TelemetryEndpoint = RegisterSetting("ROX_TELEMETRY_ENDPOINT", AllowEmpty())

	// TelemetryFrequency is the frequency at which we will report telemetry.
	TelemetryFrequency = registerDurationSetting("ROX_TELEMETRY_FREQUENCY", 10*time.Minute)

	// TelemetryStorageKey can be empty to disable telemetry collection.
	TelemetryStorageKey = RegisterSetting("ROX_TELEMETRY_STORAGE_KEY_V1", AllowEmpty())
)
