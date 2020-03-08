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
)
