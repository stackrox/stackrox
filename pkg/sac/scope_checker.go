package sac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// ScopeChecker provides a convenience wrapper around a ScopeCheckerCore.
// Client code should almost always use this type in favor of ScopeCheckerCore.
//
//go:generate mockgen-wrapper
type ScopeChecker interface {
	SubScopeChecker(keys ...ScopeKey) ScopeChecker
	IsAllowed(subScopeKeys ...ScopeKey) bool
	AllAllowed(subScopeKeyss [][]ScopeKey) bool
	ForClusterScopedObject(obj ClusterScopedObject) ScopeChecker
	ForNamespaceScopedObject(obj NamespaceScopedObject) ScopeChecker
	AccessMode(am storage.Access) ScopeChecker
	Resource(resource permissions.ResourceHandle) ScopeChecker
	ClusterID(clusterID string) ScopeChecker
	Namespace(namespace string) ScopeChecker
	Check(ctx context.Context, pred ScopePredicate) (bool, error)
	EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error)
}

type scopeChecker struct {
	core ScopeCheckerCore
}

// NewScopeChecker returns a new scope checker wrapping the given scope checker
// core.
func NewScopeChecker(core ScopeCheckerCore) ScopeChecker {
	return scopeChecker{
		core: core,
	}
}

// Core returns the ScopeCheckerCore wrapped by this scope checker.
func (c scopeChecker) Core() ScopeCheckerCore {
	return c.core
}

// SubScopeChecker returns a ScopeChecker for the given nested subscope.
func (c scopeChecker) SubScopeChecker(keys ...ScopeKey) ScopeChecker {
	curr := c.core
	for _, key := range keys {
		curr = curr.SubScopeChecker(key)
	}
	return scopeChecker{
		core: curr,
	}
}

// IsAllowed checks (in a blocking way) if access to the given (sub-)scope is allowed.
func (c scopeChecker) IsAllowed(subScopeKeys ...ScopeKey) bool {
	curr := c.core
	for _, key := range subScopeKeys {
		curr = curr.SubScopeChecker(key)
	}
	return curr.Allowed()
}

// AllAllowed checks if access to all of the given subscopes is allowed.
func (c scopeChecker) AllAllowed(subScopeKeyss [][]ScopeKey) bool {
	for _, subScopeKeys := range subScopeKeyss {
		if !c.IsAllowed(subScopeKeys...) {
			return false
		}
	}

	return true
}

// ForClusterScopedObject returns a scope checker for the subscope corresponding to the given
// cluster-scoped object.
func (c scopeChecker) ForClusterScopedObject(obj ClusterScopedObject) ScopeChecker {
	return c.SubScopeChecker(ClusterScopeKey(obj.GetClusterId()))
}

// ForNamespaceScopedObject returns a scope checker for the subscope corresponding to the given
// namespace-scoped object.
func (c scopeChecker) ForNamespaceScopedObject(obj NamespaceScopedObject) ScopeChecker {
	return c.SubScopeChecker(ClusterScopeKey(obj.GetClusterId()), NamespaceScopeKey(obj.GetNamespace()))
}

// AccessMode returns a scope checker for the subscope corresponding to the given access mode.
func (c scopeChecker) AccessMode(am storage.Access) ScopeChecker {
	return c.SubScopeChecker(AccessModeScopeKey(am))
}

// Resource returns a scope checker for the subscope corresponding to the given resource.
func (c scopeChecker) Resource(resource permissions.ResourceHandle) ScopeChecker {
	return c.SubScopeChecker(ResourceScopeKey(resource.GetResource()))
}

// ClusterID returns a scope checker for the subscope corresponding to the given cluster ID.
func (c scopeChecker) ClusterID(clusterID string) ScopeChecker {
	return c.SubScopeChecker(ClusterScopeKey(clusterID))
}

// Namespace returns a scope checker for the subscope corresponding to the given namespace.
func (c scopeChecker) Namespace(namespace string) ScopeChecker {
	return c.SubScopeChecker(NamespaceScopeKey(namespace))
}

// Check checks the given predicate in this scope.
func (c scopeChecker) Check(_ context.Context, pred ScopePredicate) (bool, error) {
	return pred.Allowed(c), nil
}

// EffectiveAccessScope returns underlying effective access scope.
func (c scopeChecker) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return c.core.EffectiveAccessScope(resource)
}
