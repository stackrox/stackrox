package k8sutil

// NamespacedObject is the minimal interface for Kubernetes objects with a name and a namespace.
type NamespacedObject interface {
	GetNamespace() string
	GetName() string
}

// NSObjRef can be used to maintain a reference to a namespaced object in Kubernetes, identified through its namespace
// and name. This type is safe to be used as a key in a map.
// Note that this DOES NOT allow to distinguish between objects with different types but equal names.
type NSObjRef struct {
	Namespace, Name string
}

// RefOf obtains a NSObjRef for the given object.
func RefOf(obj NamespacedObject) NSObjRef {
	return NSObjRef{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}
