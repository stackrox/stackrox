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
)
