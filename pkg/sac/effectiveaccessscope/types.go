package effectiveaccessscope

// ClusterForSAC is the minimum interface a cluster object has to satisfy
// to be used in effective access scope computation
type ClusterForSAC interface {
	GetID() string
	GetName() string
	GetLabels() map[string]string
}

// NamespaceForSAC is the minimum interface a namespace object has to satisfy
// to be used in effective access scope computation
type NamespaceForSAC interface {
	GetID() string
	GetName() string
	GetClusterName() string
	GetLabels() map[string]string
}
