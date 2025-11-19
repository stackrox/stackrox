package kubernetes

const (
	// TLSCABundleConfigMapName is the name of a ConfigMap that Sensor creates at runtime
	// to store the internal CA certificates trusted by Central. This ConfigMap is consumed
	// by the Operator to update the ValidatingWebhookConfiguration's caBundle.
	TLSCABundleConfigMapName = "tls-ca-bundle"

	// TLSCABundleKey is the key for the CA bundle in the TLSCABundleConfigMap.
	TLSCABundleKey = "ca-bundle.pem"
)
