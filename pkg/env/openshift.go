package env

var (
	// OpenshiftAPI specifies whether the k8s listener should talk via the openshift API.
	// ROX_OPENSHIFT_API is referenced in the installation files, please be cautious when removing.
	OpenshiftAPI = RegisterBooleanSetting("ROX_OPENSHIFT_API", false)

	// EnableOpenShiftAuth specifies whether authentication via OpenShift's
	// built-in OAuth server shall be enabled in Central. Note that just
	// switching this on is not enough because extra steps are required to
	// configure Central as an OAuth client.
	EnableOpenShiftAuth = RegisterBooleanSetting("ROX_ENABLE_OPENSHIFT_AUTH", false)

	// Openshift specifies whether Openshift is the orchestrator.
	Openshift = RegisterBooleanSetting("ROX_OPENSHIFT", false)
)
