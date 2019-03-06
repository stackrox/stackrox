package env

const (
	defaultOpenshiftAPI = `false`
)

var (
	// OpenshiftAPI is whether or not the k8s listener should talk via the openshift API
	// ROX_OPENSHIFT_API is referenced in the deploy files, please look before removing
	OpenshiftAPI = RegisterSetting("ROX_OPENSHIFT_API", WithDefault(defaultOpenshiftAPI), AllowEmpty())
)
