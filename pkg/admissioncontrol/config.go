package admissioncontrol

const (
	// ConfigMapName is the name of the config map that stores the admission controller configuration.
	ConfigMapName = `admission-control`

	// ConfigGZDataKey is the key in the config map under which the serialized dynamic cluster config is stored.
	ConfigGZDataKey = `config.pb.gz`

	// PoliciesGZDataKey is the key in the config map under which the serialized enforced deploy-time policies are
	// stored.
	PoliciesGZDataKey = `policies.pb.gz`

	// LastUpdateTimeDataKey is the key in the config map under which the "last updated" timestamp is stored.
	// This is stored as a data member instead of an annotation in order to allow accessing this also via a file
	// mount.
	LastUpdateTimeDataKey = `last-update-time`

	// CacheVersionDataKey is the key in the config map under which the "cache version" is stored.
	// A change of this value signals the admission controller to flush its internal caches.
	CacheVersionDataKey = `cache-version`

	// CentralEndpointDataKey is the key in the config map under which the central endpoint is stored.
	CentralEndpointDataKey = `central-endpoint`

	// ClusterIDDataKey is the key in the config map under which the cluster ID is stored.
	ClusterIDDataKey = `cluster-id`
)
