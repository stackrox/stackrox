package permissions

// Resource is a string representation of an exposed set of API endpoints (services).
type Resource string

// GetResource returns the name of the resource.
func (r Resource) GetResource() Resource {
	return r
}

// ResourceScope is used to indicate the scope of a resource.
type ResourceScope int

const (
	// GlobalScope means the resource is global only.
	GlobalScope ResourceScope = iota
	// ClusterScope means the resource exists in a cluster scope.
	ClusterScope
	// NamespaceScope means the resource exists in a cluster and namespace determined scope.
	NamespaceScope
)

// ResourceMetadata contains metadata about a resource.
type ResourceMetadata struct {
	Resource
	Scope ResourceScope
}

// GetResource returns the resource for this metadata object.
func (m ResourceMetadata) GetResource() Resource {
	return m.Resource
}

// GetScope returns the resource scope for this metadata object.
func (m ResourceMetadata) GetScope() ResourceScope {
	return m.Scope
}

// String returns a string representation of the resource for this metadata object.
func (m ResourceMetadata) String() string {
	return string(m.Resource)
}

// ResourceHandle allows referring to a resource, without having to specify whether it is a Resource
// or a ResourceMetadata object.
type ResourceHandle interface {
	GetResource() Resource
}
