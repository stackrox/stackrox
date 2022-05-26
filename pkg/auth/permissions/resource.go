package permissions

import "github.com/stackrox/rox/generated/storage"

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
	// ReplacingResource may be used when the given Resource has a replacing equivalent, i.e. during deprecation.
	// The ReplacingResource will be used during SAC checks for the resource, essentially allowing access if either
	// access to the old Resource is allowed OR to the ReplacingResource.
	ReplacingResource *ResourceMetadata
	Scope             ResourceScope
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
	// Replacing resources _may_ have a different ResourceScope than the initial resource.
	// This _may_ broaden the scope, i.e. when the current resource is namespace scoped and the replacing
	// resource is cluster scoped.
	if m.ReplacingResource != nil && m.ReplacingResource.GetScope() < m.Scope {
		return m.ReplacingResource.GetScope()
	}
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

// IsPermittedBy returns whether the ResourceMetadata is contained within the map
// with at least the specified storage.Access.
// Note: This will take replacing resources into account.
func (m ResourceMetadata) IsPermittedBy(resourceAccessMap map[string]storage.Access, access storage.Access) bool {
	if resourceAccessMap[string(m.GetResource())] >= access {
		return true
	} else if m.ReplacingResource != nil &&
		// Right now, we are not taking multiple replacing resources into account, i.e. Resource A has replacing
		// resource "Resource B", which also has a replacing resource "Resource C".
		resourceAccessMap[string(m.ReplacingResource.GetResource())] >= access {
		return true
	}
	return false
}
