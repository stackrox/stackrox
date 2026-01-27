package env

import "os"

var (
	// ContinuousProfiling indicates if continuous profiling is enabled
	ContinuousProfiling = RegisterBooleanSetting("ROX_CONTINUOUS_PROFILING", false)

	// ContinuousProfilingServerAddress defines the server address for the continuous profiling
	ContinuousProfilingServerAddress = RegisterSetting("ROX_CONTINUOUS_PROFILING_SERVER_ADDRESS", WithDefault("http://pyroscope.stackrox.svc.cluster.local.:4040"))

	// ContinuousProfilingBasicAuthUser defines the http basic auth user
	ContinuousProfilingBasicAuthUser = RegisterSetting("ROX_CONTINUOUS_PROFILING_BASIC_AUTH_USER")

	// ContinuousProfilingBasicAuthPassword defines the http basic auth password
	ContinuousProfilingBasicAuthPassword = RegisterSetting("ROX_CONTINUOUS_PROFILING_BASIC_AUTH_PASSWORD")

	// ContinuousProfilingAppName defines the AppName used to send the profiles
	ContinuousProfilingAppName = RegisterSetting("ROX_CONTINUOUS_PROFILING_APP_NAME", WithDefault(os.Getenv("POD_NAME")))

	// ContinuousProfilingLabels defines additional labels/tags to attach to profiling data sent to Pyroscope.
	// Format: Comma-separated list of key=value pairs (e.g., "env=production,region=us-east,team=security")
	//
	// Parsing behavior:
	// - Whitespace around keys, values, and commas is trimmed
	// - Empty entries (consecutive commas) are ignored
	// - Values can contain equals signs (e.g., "token=abc=123" is valid)
	// - Both keys and values must be non-empty after trimming
	//
	// Error handling:
	// - An invalid entry is logged as an error but doesn't prevent profiler initialization
	// - If parsing fails, no labels are set
	//
	// Examples:
	//   Valid:   "app=central,env=prod"
	//   Valid:   " key1 = value1 , key2 = value2 "  (whitespace is trimmed)
	//   Valid:   "token=abc=123"                    (values can contain '=')
	//   Invalid: "key="                              (empty value)
	//   Invalid: "=value"                            (empty key)
	//   Invalid: "noequals"                          (missing '=' separator)
	ContinuousProfilingLabels = RegisterSetting("ROX_CONTINUOUS_PROFILING_LABELS")
)
