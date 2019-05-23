package sac

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

// ForResourceHelper is a helper for querying access scopes related to a resource.
type ForResourceHelper struct {
	resource permissions.Resource
}

// ForResource returns a helper for querying access scopes related to the given resource.
func ForResource(resource permissions.Resource) ForResourceHelper {
	return ForResourceHelper{
		resource: resource,
	}
}

// ScopeChecker returns the scope checker for accessing the given resource in the specified way.
func (h ForResourceHelper) ScopeChecker(ctx context.Context, am storage.Access, keys ...ScopeKey) ScopeChecker {
	return GlobalAccessScopeChecker(ctx).AccessMode(am).Resource(h.resource).SubScopeChecker(keys...)
}

// AccessAllowed checks if in the given context, we have access of the specified kind to the resource or
// a subscope thereof.
func (h ForResourceHelper) AccessAllowed(ctx context.Context, am storage.Access, keys ...ScopeKey) (bool, error) {
	return h.ScopeChecker(ctx, am, keys...).Allowed(ctx)
}

// ReadAllowed checks if in the given context, we have read access to the resource or a subscope thereof.
func (h ForResourceHelper) ReadAllowed(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowed(ctx, storage.Access_READ_ACCESS, keys...)
}

// WriteAllowed checks if in the given context, we have write access to the resource or a subscope thereof.
func (h ForResourceHelper) WriteAllowed(ctx context.Context, keys ...ScopeKey) (bool, error) {
	return h.AccessAllowed(ctx, storage.Access_READ_WRITE_ACCESS, keys...)
}

// MustCreateSearchHelper creates and returns a search helper with the given options, or panics if the
// search helper could not be created.
func (h ForResourceHelper) MustCreateSearchHelper(options search.OptionsMap, flavor SearchHelperFlavor) SearchHelper {
	searchHelper, err := NewSearchHelper(h.resource, options, flavor)
	utils.Must(err)
	return searchHelper
}

// ReadAllowedForClusterNSScopes checks if the principal from the given context is allowed to read an object
// with the given Cluster/NS scope values.
func (h ForResourceHelper) ReadAllowedForClusterNSScopes(ctx context.Context, clusterNSScopesMap map[string]string) (bool, error) {
	allScopeKeys := make([][]ScopeKey, 0, len(clusterNSScopesMap))

	for _, clusterNSScope := range clusterNSScopesMap {
		allScopeKeys = append(allScopeKeys, ParseClusterNSScopeString(clusterNSScope))
	}

	return h.ScopeChecker(ctx, storage.Access_READ_ACCESS).AnyAllowed(ctx, allScopeKeys)
}
