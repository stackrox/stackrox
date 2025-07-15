package securedcluster

const (
	// CABundleConfigMapName is the name of a ConfigMap that Sensor creates at runtime
	// to store the CA certificates retrieved from Central.
	CABundleConfigMapName = "tls-ca-bundle"
)
