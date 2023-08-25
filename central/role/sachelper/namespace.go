package sachelper

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// listNamespaceNamesInScope consolidates the list of names of namespaces in the cluster matching
// the requested ID and allowed by user scopes associated with the requested resources
// and access level.
// - If one of the allowed scopes is unrestricted for the requested cluster, then the string set
// is returned empty and the returned boolean is true.
// - If no allowed scope is unrestricted for the requested cluster, the string set contains
// the names of the namespaces allowed by the user scopes associated with the requested resources,
// and the returned boolean is false.
func listNamespaceNamesInScope(
	ctx context.Context,
	clusterID string,
	resourcesWithAccess []permissions.ResourceWithAccess,
) (set.StringSet, bool, error) {
	noNamespaces := set.NewStringSet()
	namespacesInScope := set.NewStringSet()
	for _, r := range resourcesWithAccess {
		scope, err := getRequesterScopeForReadPermission(ctx, r)
		if err != nil {
			return noNamespaces, partialAccess, err
		}
		if scope == nil || scope.State == effectiveaccessscope.Excluded {
			continue
		}
		if scope.State == effectiveaccessscope.Included {
			return noNamespaces, fullAccess, nil
		}
		clusterScope := scope.GetClusterByID(clusterID)
		if clusterScope == nil || clusterScope.State == effectiveaccessscope.Excluded {
			continue
		}
		if clusterScope.State == effectiveaccessscope.Included {
			return noNamespaces, fullAccess, nil
		}
		for namespace, namespaceScope := range clusterScope.Namespaces {
			if namespaceScope == nil || namespaceScope.State == effectiveaccessscope.Excluded {
				continue
			}
			namespacesInScope.Add(namespace)
		}
	}
	return namespacesInScope, partialAccess, nil
}

func getNamespacesOptionsMap() search.OptionsMap {
	return schema.NamespacesSchema.OptionsMap
}
