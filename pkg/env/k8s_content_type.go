package env

var (
	// KubernetesClientContentType overrides the default Kubernetes content type of protobuf
	KubernetesClientContentType = RegisterSetting("ROX_K8S_CLIENT_CONTENT_TYPE")
)
