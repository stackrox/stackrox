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

// KeyForNSScopedObj returns a scope key slice for the access scope of the given namespace-scoped object.
func KeyForNSScopedObj(obj NamespaceScopedObject) []ScopeKey {
	return []ScopeKey{ClusterScopeKey(obj.GetClusterId()), NamespaceScopeKey(obj.GetNamespace())}
}
