package env

import "time"

// TelemetrySelfManagedURL is the configuration URL for self-managed instances.
// TODO(ROX-17726): Set default URL for self-managed installations use, and
// update the code to not ignore this URL.
const TelemetrySelfManagedURL = "hardcoded"

var (
	// Phone-Home telemetry variables:

	// TelemetryEndpoint is the URL to send telemetry to.
	TelemetryEndpoint = RegisterSetting("ROX_TELEMETRY_ENDPOINT",
		WithDefault("https://console.redhat.com/connections/api"), AllowEmpty())

	// TelemetryConfigURL to retrieve the telemetry configuration from.
	// AllowEmpty() allows for disabling the downloading, and for providing a
	// custom key to release binary versions.
	TelemetryConfigURL = RegisterSetting("ROX_TELEMETRY_CONFIG_URL", AllowEmpty(), WithDefault(TelemetrySelfManagedURL))

	// TelemetryFrequency is the frequency at which we will report telemetry.
	TelemetryFrequency = registerDurationSetting("ROX_TELEMETRY_FREQUENCY", 10*time.Minute)

	// TelemetryStorageKey can be empty to disable telemetry collection.
	TelemetryStorageKey = RegisterSetting("ROX_TELEMETRY_STORAGE_KEY_V1", AllowEmpty())

	// ExecutionEnvironment specifies the context of the roxctl run. For example: "GHA" or "Tekton".
	// The value will be sent within the Rh-Execution-Environment HTTP header.
	ExecutionEnvironment = RegisterSetting("ROX_EXECUTION_ENV", AllowEmpty())
)
