package sac

import (
	"context"
	"slices"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// ForResourceHelper is a helper for querying access scopes related to a resource.
type ForResourceHelper struct {
	resourceMD permissions.ResourceMetadata
}

// ForResource returns a helper for querying access scopes related to the given resource.
func ForResource(resourceMD permissions.ResourceMetadata) ForResourceHelper {
	return ForResourceHelper{
		resourceMD: resourceMD,
	}
}

// ScopeChecker returns the scope checker for accessing the given resource in the specified way.
func (h ForResourceHelper) ScopeChecker(ctx context.Context, am storage.Access, keys ...ScopeKey) ScopeChecker {
	resourceScopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(am).Resource(
		h.resourceMD).SubScopeChecker(keys...)

	if h.resourceMD.GetReplacingResource() == nil {
		return resourceScopeChecker
	}
	// Conditionally create a OR scope checker if a replacing resource is given. This way we check access to either
	// the old resource OR the replacing resource, keeping backwards-compatibility.
	return NewOrScopeChecker(
		resourceScopeChecker,
		GlobalAccessScopeChecker(ctx).AccessMode(am).
			Resource(h.resourceMD.ReplacingResource).SubScopeChecker(keys...))
}

// AccessAllowed checks if in the given context, we have access of the specified kind to the resource or
// a subscope thereof.
func (h ForResourceHelper) AccessAllowed(ctx context.Context, am storage.Access, keys ...ScopeKey) (bool, error) {
	return h.ScopeChecker(ctx, am, keys...).IsAllowed(), nil
}

// ReadAllowed checks if in the given context, we have read access to the resource or a subscope thereof.
func (h ForResourceHelper) ReadAllowed(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowed(ctx, storage.Access_READ_ACCESS, keys...)
}

// WriteAllowed checks if in the given context, we have write access to the resource or a subscope thereof.
func (h ForResourceHelper) WriteAllowed(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowed(ctx, storage.Access_READ_WRITE_ACCESS, keys...)
}

func (h ForResourceHelper) HasGlobalRead(ctx context.Context) (bool, error) {
	return h.ReadAllowed(ctx)
}

func (h ForResourceHelper) HasGlobalWrite(ctx context.Context) (bool, error) {
	return h.WriteAllowed(ctx)
}

func (h ForResourceHelper) FilterAccessibleNamespaces(
	ctx context.Context,
	accessMode storage.Access,
	namespaces []*storage.NamespaceMetadata,
) ([]*storage.NamespaceMetadata, error) {
	hasGlobalAccess, err := h.AccessAllowed(ctx, accessMode)
	if err != nil {
		return nil, err
	}
	if hasGlobalAccess {
		return slices.Clone(namespaces), nil
	}
	scopeChecker := h.ScopeChecker(ctx, accessMode)
	accessScopeTree, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
		Resource: h.resourceMD,
		Access:   accessMode,
	})
	if err != nil {
		return nil, err
	}
	if accessScopeTree.State == effectiveaccessscope.Included {
		return slices.Clone(namespaces), nil
	}
	filtered := make([]*storage.NamespaceMetadata, 0, len(namespaces))
	for _, ns := range namespaces {
		clusterSubTree := accessScopeTree.GetClusterByID(ns.GetClusterId())
		if clusterSubTree == nil || clusterSubTree.State == effectiveaccessscope.Excluded {
			continue
		}
		if h.resourceMD.Scope == permissions.ClusterScope || clusterSubTree.State == effectiveaccessscope.Included {
			filtered = append(filtered, ns)
			continue
		}
		namespaceSubTree := clusterSubTree.Namespaces[ns.GetName()]
		if namespaceSubTree == nil || namespaceSubTree.State == effectiveaccessscope.Excluded {
			continue
		}
		filtered = append(filtered, ns)
	}
	return filtered, nil
}
