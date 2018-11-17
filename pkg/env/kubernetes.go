package env

const (
	defaultNamespace = `stackrox`
)

// These environment variables are referenced within the deployment files.
// Please check them before deleting
var (
	// Namespace is the namespace in which sensors and benchmark services are deployed (k8s).
	Namespace = NewSetting("ROX_NAMESPACE", WithDefault(defaultNamespace))
)
