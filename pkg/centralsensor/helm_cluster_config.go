package centralsensor

const (
	// HelmManagedClusterMetadataKey is the key to indicate by both sensor and central that the cluster will be
	// Helm-managed. The only value to be used for this metadata key is "true".
	HelmManagedClusterMetadataKey = `Rox-Helm-Managed-Cluster`
)
