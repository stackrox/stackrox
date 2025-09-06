package annotations

const (
	// ConfigHashAnnotation is a level-based annotation that is persisted in the Central and
	// SecuredClusters Deployments and DaemonSets pod templates, to drive restarts when config changes.
	// Currently only the CA certificate is hashed.
	ConfigHashAnnotation = "app.stackrox.io/config-hash"
)
