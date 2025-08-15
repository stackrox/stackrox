package securedcluster

// TLS secret names used by SecuredCluster services.
// These secret names follow a different convention than our legacy secrets (e.g. sensor-tls, scanner-tls etc.),
// so that they can both exist in parallel. This is in order to not create conflicts with automations of existing
// deployments that might provide those legacy secrets.
const (
	SensorTLSSecretName           = "tls-cert-sensor"             // #nosec G101 -- Resource names, not credentials
	CollectorTLSSecretName        = "tls-cert-collector"          // #nosec G101 -- Resource names, not credentials
	AdmissionControlTLSSecretName = "tls-cert-admission-control"  // #nosec G101 -- Resource names, not credentials
	ScannerTLSSecretName          = "tls-cert-scanner"            // #nosec G101 -- Resource names, not credentials
	ScannerDbTLSSecretName        = "tls-cert-scanner-db"         // #nosec G101 -- Resource names, not credentials
	ScannerV4IndexerTLSSecretName = "tls-cert-scanner-v4-indexer" // #nosec G101 -- Resource names, not credentials
	ScannerV4DbTLSSecretName      = "tls-cert-scanner-v4-db"      // #nosec G101 -- Resource names, not credentials
)
