package common

// Well-known secret names used by the operator
const (
	// SensorTLSSecretName is the name of the sensor TLS secret
	SensorTLSSecretName = "tls-cert-sensor" // #nosec G101 -- This is a resource name, not a credential
)
