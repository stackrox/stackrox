package sac

// ClusterScopedObject is a superinterface for all protobuf-generated objects that carry a cluster ID.
type ClusterScopedObject interface {
	GetClusterId() string
}

// NamespaceScopedObject is a superinterface for all protobuf-generated objects that carry a cluster ID
// and a namespace.
type NamespaceScopedObject interface {
	ClusterScopedObject
	GetNamespace() string
}
