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
	Allowed(ctx context.Context, subScopeKeys ...ScopeKey) (bool, error)
	AllAllowed(ctx context.Context, subScopeKeyss [][]ScopeKey) (bool, error)
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

// TryAllowed checks (in a non-blocking way) if access to the given (sub-)scope is allowed.
func (c scopeChecker) TryAllowed(subScopeKeys ...ScopeKey) bool {
	curr := c.core
	for _, key := range subScopeKeys {
		curr = curr.SubScopeChecker(key)
	}
	return curr.Allowed()
}

// Allowed checks (in a blocking way) if access to the given (sub-)scope is allowed.
func (c scopeChecker) Allowed(ctx context.Context, subScopeKeys ...ScopeKey) (bool, error) {
	curr := c.core
	for _, key := range subScopeKeys {
		curr = curr.SubScopeChecker(key)
	}
	return curr.Allowed(), nil
}

// AllAllowed checks if access to all of the given subscopes is allowed.
func (c scopeChecker) AllAllowed(ctx context.Context, subScopeKeyss [][]ScopeKey) (bool, error) {
	for _, subScopeKeys := range subScopeKeyss {
		if !c.TryAllowed(subScopeKeys...) {
			return false, nil
		}
	}

	return true, nil
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
func (c scopeChecker) Check(ctx context.Context, pred ScopePredicate) (bool, error) {
	return pred.TryAllowed(c), nil
}

// EffectiveAccessScope returns underlying effective access scope.
func (c scopeChecker) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return c.core.EffectiveAccessScope(resource)
}
