package env

const (
	defaultOpenshiftAPI = `false`
)

var (
	// OpenshiftAPI is whether or not the k8s listener should talk via the openshift API
	OpenshiftAPI = NewSetting("ROX_OPENSHIFT_API", WithDefault(defaultOpenshiftAPI), AllowEmpty())
)
