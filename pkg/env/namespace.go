package env

var (
	// Namespace specifies the namespace that the pod is in via the downward API or it defaults to stackrox
	Namespace = RegisterSetting("ROX_NAMESPACE", WithDefault("stackrox"))
)
