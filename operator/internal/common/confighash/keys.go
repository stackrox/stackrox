package confighash

const (
	// AnnotationKey is a level-based annotation that is persisted in the Central and
	// SecuredCluster Deployments and DaemonSets pod templates, to trigger pod restarts when config changes.
	// Currently only the CA certificate is hashed.
	AnnotationKey = "app.stackrox.io/config-hash"
)
