package env

import "time"

var (
	// InitialTelemetryEnabledEnv indicates whether StackRox was installed with telemetry enabled.  This flag is
	// overridden by the telemetry configuration in the database  Defaults to false here and true in the install process
	// so that it will default to on for new installations and off for old installations
	InitialTelemetryEnabledEnv = RegisterBooleanSetting("ROX_INIT_TELEMETRY_ENABLED", false)

	// TelemetryEndpoint is the endpoint to which to send telemetry data.
	TelemetryEndpoint = RegisterSetting("ROX_TELEMETRY_ENDPOINT", AllowEmpty())

	// TelemetryFrequency is the frequency at which we will report telemetry
	TelemetryFrequency = registerDurationSetting("ROX_TELEMETRY_FREQUENCY", 24*time.Hour)

	// AmplitudeAPIKey can be empty to disable marketing telemetry collection
	AmplitudeAPIKey = RegisterSetting("AMPLITUDE_API_KEY", AllowEmpty())
)
