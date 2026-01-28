package env

var (
	// KubernetesClientQPS defines the maximum queries per second (QPS) to the
	// Kubernetes API server from the client-go client. The default k8s client-go
	// value is 5, which can cause client-side throttling in high-activity environments.
	// Setting this higher avoids the "client-side throttling, not priority and fairness"
	// warning messages.
	KubernetesClientQPS = RegisterFloatSetting("ROX_KUBERNETES_CLIENT_QPS", 50).WithMinimum(1)

	// KubernetesClientBurst defines the maximum burst for throttle to the
	// Kubernetes API server from the client-go client. This allows temporary
	// bursts above the QPS rate. The default k8s client-go value is 10.
	KubernetesClientBurst = RegisterIntegerSetting("ROX_KUBERNETES_CLIENT_BURST", 100).WithMinimum(1)
)
