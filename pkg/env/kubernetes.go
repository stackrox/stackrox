package env

const (
	defaultNamespace = `stackrox`
)

var (
	// Namespace is the namespace in which sensors and benchmark services are deployed (k8s).
	Namespace = NewSetting("ROX_NAMESPACE", WithDefault(defaultNamespace))
)
