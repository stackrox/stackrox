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
	// legacyAuthForSAC is a tri-state bool determining whether legacy auth for SAC is forced on or off. If false,
	// no legacy auth for SAC is performed (only affects globally-scoped resources). If true, legacy auth for SAC
	// (at the global scope) is performed even for non-globally scoped resources. If `nil`, the default behavior is used
	// (i.e., performing legacy auth for globally-scoped resources, and not performing it for resources with cluster
	// or namespace scopes).
	legacyAuthForSAC *bool
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

// PerformLegacyAuthForSAC checks whether legacy authorizers should be enforced even if SAC is enabled.
func (m ResourceMetadata) PerformLegacyAuthForSAC() bool {
	if m.legacyAuthForSAC != nil {
		return *m.legacyAuthForSAC
	}
	return m.Scope == GlobalScope
}

// ResourceHandle allows referring to a resource, without having to specify whether it is a Resource
// or a ResourceMetadata object.
type ResourceHandle interface {
	GetResource() Resource
}

// WithLegacyAuthForSAC returns a resource metadata that instructs the legacy auth handler to either force or force
// skip legacy auth for SAC.
func WithLegacyAuthForSAC(md ResourceMetadata, use bool) ResourceMetadata {
	md.legacyAuthForSAC = &use
	return md
}
