package env

const (
	defaultNamespace = `stackrox`
)

var (
	// Namespace is the namespace in which sensors and benchmark services are deployed (k8s).
	Namespace = NewSetting("ROX_NAMESPACE", WithDefault(defaultNamespace))
	// ServiceAccount designates the account under which sensors and benchmarks operate (k8s).
	ServiceAccount = NewSetting("ROX_SERVICE_ACCOUNT")
	// ImagePullSecrets are secrets used for pulling images (k8s).
	ImagePullSecrets = NewSetting("ROX_IMAGE_PULL_CONFIG")
)
