package env

const (
	defaultOpenshiftAPI = "false"
)

var (
	// OpenshiftAPI is whether or not the k8s listener should talk via the openshift API
	// ROX_OPENSHIFT_API is referenced in the deploy files, please look before removing
	OpenshiftAPI = RegisterSetting("ROX_OPENSHIFT_API", WithDefault(defaultOpenshiftAPI), AllowEmpty())

	// EnableOpenShiftAuth specifies whether authentication via OpenShift's
	// built-in OAuth server shall be enabled in Central. Note that just
	// switching this on is not enough because extra steps are required to
	// configure Central as an OAuth client.
	EnableOpenShiftAuth = RegisterBooleanSetting("ROX_ENABLE_OPENSHIFT_AUTH", false)
)
