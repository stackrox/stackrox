package env

import "time"

var (
	// InitialTelemetryEnabledEnv indicates whether StackRox was installed with telemetry enabled.  This flag is
	// overridden by the telemetry configuration in the database  Defaults to false here and true in the install process
	// so that it will default to on for new installations and off for old installations
	InitialTelemetryEnabledEnv = RegisterBooleanSetting("ROX_INIT_TELEMETRY_ENABLED", false)

	// Phone-Home telemetry variables:

	// TelemetryEndpoint is the endpoint to which to send telemetry data.
	TelemetryEndpoint = RegisterSetting("ROX_TELEMETRY_ENDPOINT", AllowEmpty())

	// TelemetryFrequency is the frequency at which we will report telemetry.
	TelemetryFrequency = registerDurationSetting("ROX_TELEMETRY_FREQUENCY", 10*time.Minute)

	// TelemetryStorageKey can be empty to disable telemetry collection.
	TelemetryStorageKey = RegisterSetting("ROX_TELEMETRY_STORAGE_KEY_V1", AllowEmpty())
)
