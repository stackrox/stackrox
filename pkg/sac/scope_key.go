package sac

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// ScopeKind identifies the kind of an access scope.
type ScopeKind int

const (
	// GlobalScopeKind identifies the global scope. This scope does not have a key.
	GlobalScopeKind ScopeKind = iota
	// AccessModeScopeKind identifies the access mode scope (read or read/write).
	AccessModeScopeKind
	// ResourceScopeKind identifies the resource scope.
	ResourceScopeKind
	// ClusterScopeKind identifies the cluster scope.
	ClusterScopeKind
	// NamespaceScopeKind identifies the namespace scope.
	NamespaceScopeKind
)

// ScopeKey is a common superinterface for all access scope keys.
// This interface can only be implemented by types in this package. The intention is to
// ensure strong typing for every kind of scope key.
type ScopeKey interface {
	fmt.Stringer
	ScopeKind() ScopeKind

	isScopeKey()
}

// AccessModeScopeKey is the scope key for the access mode scope.
type AccessModeScopeKey storage.Access

func (AccessModeScopeKey) isScopeKey() {}

// ScopeKind returns `AccessModeScopeKind`.
func (AccessModeScopeKey) ScopeKind() ScopeKind {
	return AccessModeScopeKind
}

// String returns a string representation for this access scope key.
func (k AccessModeScopeKey) String() string {
	return storage.Access(k).String()
}

// Verb returns a string version of this access scope suitable for sending to third party auth plugins
func (k AccessModeScopeKey) Verb() string {
	switch storage.Access(k) {
	case storage.Access_READ_ACCESS:
		return "view"
	case storage.Access_READ_WRITE_ACCESS:
		return "edit"
	default:
		return ""
	}
}

// AccessModeScopeKeys wraps the given access modes in a scope key slice.
func AccessModeScopeKeys(ams ...storage.Access) []AccessModeScopeKey {
	keys := make([]AccessModeScopeKey, len(ams))
	for i, am := range ams {
		keys[i] = AccessModeScopeKey(am)
	}
	return keys
}

// ResourceScopeKey is the scope key for the resource scope.
type ResourceScopeKey permissions.Resource

func (ResourceScopeKey) isScopeKey() {}

// ScopeKind returns `ResourceScopeKind`.
func (ResourceScopeKey) ScopeKind() ScopeKind {
	return ResourceScopeKind
}

// String returns a string representation for this access scope key.
func (k ResourceScopeKey) String() string {
	return string(k)
}

// ResourceScopeKeys wraps the given resources in a scope key slice.
// Note: The returned scope keys _may_ be greater than the given resources,
// since replacing resources will be taken into account and conditionally added
// to the returned scope keys.
// This should be fine, as the ResourceScopeKeys is used in contexts which do not
// specifically require a fixed length based on the number of permissions.ResourceHandle.
func ResourceScopeKeys(resources ...permissions.ResourceHandle) []ResourceScopeKey {
	keys := make([]ResourceScopeKey, 0, len(resources))
	for _, resource := range resources {
		keys = append(keys, ResourceScopeKey(resource.GetResource()))
		if resource.GetReplacingResource() != nil {
			keys = append(keys, ResourceScopeKey(*resource.GetReplacingResource()))
		}
	}
	return keys
}

// ClusterScopeKey is the scope key for the cluster scope.
type ClusterScopeKey string

func (ClusterScopeKey) isScopeKey() {}

// ScopeKind returns `ClusterScopeKind`.
func (ClusterScopeKey) ScopeKind() ScopeKind {
	return ClusterScopeKind
}

// String returns a string representation for this access scope key.
func (k ClusterScopeKey) String() string {
	return string(k)
}

// ClusterScopeKeys wraps the given cluster IDs in a scope key slice.
func ClusterScopeKeys(clusterIDs ...string) []ClusterScopeKey {
	keys := make([]ClusterScopeKey, len(clusterIDs))
	for i, clusterID := range clusterIDs {
		keys[i] = ClusterScopeKey(clusterID)
	}
	return keys
}

// NamespaceScopeKey is the scope key for the namespace scope.
type NamespaceScopeKey string

func (NamespaceScopeKey) isScopeKey() {}

// ScopeKind returns `NamespaceScopeKind`.
func (NamespaceScopeKey) ScopeKind() ScopeKind {
	return NamespaceScopeKind
}

// String returns a string representation for this access scope key.
func (k NamespaceScopeKey) String() string {
	return string(k)
}

// NamespaceScopeKeys wraps the given namespaces in a scope key slice.
func NamespaceScopeKeys(namespaces ...string) []NamespaceScopeKey {
	keys := make([]NamespaceScopeKey, len(namespaces))
	for i, namespace := range namespaces {
		keys[i] = NamespaceScopeKey(namespace)
	}
	return keys
}

// ScopePredicate is a common interface for all objects that can be interpreted as an expression over scopes.
type ScopePredicate interface {
	Allowed(sc ScopeChecker) bool
}

// ScopeSuffix is a predicate that checks if the given scope suffix (relative to the checker) is allowed.
type ScopeSuffix []ScopeKey

// Allowed implements the ScopePredicate interface.
func (i ScopeSuffix) Allowed(sc ScopeChecker) bool {
	return sc.IsAllowed(i...)
}

// AnyScope is a scope predicate that evaluates to Allowed if any of the given scopes is allowed.
type AnyScope []ScopePredicate

// Allowed implements the ScopePredicate interface.
func (p AnyScope) Allowed(sc ScopeChecker) bool {
	for _, pred := range p {
		if pred.Allowed(sc) {
			return true
		}
	}
	return false
}

// AllScopes is a scope predicate that evaluates to Allowed if all of the given scopes are allowed.
type AllScopes []ScopePredicate

// Allowed implements the ScopePredicate interface.
func (p AllScopes) Allowed(sc ScopeChecker) bool {
	for _, pred := range p {
		if !pred.Allowed(sc) {
			return false
		}
	}
	return true
}
