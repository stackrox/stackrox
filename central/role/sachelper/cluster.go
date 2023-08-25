package sachelper

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// listClusterIDsInScope consolidates the list of cluster IDs in the user scopes associated
// with the requested resources and access level.
// - If one of the allowed scopes is unrestricted, then the string set is returned empty
// and the returned boolean is true.
// - If no allowed scope is unrestricted, the string set contains the cluster IDs allowed by
// the user scopes associated with the requested resources, and the returned boolean is false.
func listClusterIDsInScope(
	ctx context.Context,
	resourcesWithAccess []permissions.ResourceWithAccess,
) (set.StringSet, bool, error) {
	clusterIDsInScope := set.NewStringSet()
	for _, r := range resourcesWithAccess {
		scope, err := getRequesterScopeForReadPermission(ctx, r)
		if err != nil {
			return set.NewStringSet(), partialAccess, err
		}
		if scope == nil || scope.State == effectiveaccessscope.Excluded {
			continue
		}
		if scope.State == effectiveaccessscope.Included {
			return set.NewStringSet(), fullAccess, nil
		}
		clusterIDs := scope.GetClusterIDs()
		for _, clusterID := range clusterIDs {
			if clusterNode := scope.GetClusterByID(clusterID); clusterNode != nil &&
				clusterNode.State != effectiveaccessscope.Excluded {
				clusterIDsInScope.Add(clusterID)
			}
		}
	}
	return clusterIDsInScope, partialAccess, nil
}

func hasClusterIDInScope(
	ctx context.Context,
	clusterID string,
	resourcesWithAccess []permissions.ResourceWithAccess,
) (bool, bool, error) {
	for _, r := range resourcesWithAccess {
		scope, err := getRequesterScopeForReadPermission(ctx, r)
		if err != nil {
			return false, false, err
		}
		if scope == nil || scope.State == effectiveaccessscope.Excluded {
			continue
		}
		if scope.State == effectiveaccessscope.Included {
			return false, fullAccess, nil
		}
		clusterSubTree := scope.GetClusterByID(clusterID)
		if clusterSubTree != nil && clusterSubTree.State != effectiveaccessscope.Excluded {
			return true, partialAccess, nil
		}
	}
	return false, partialAccess, nil
}

func getClustersOptionsMap() search.OptionsMap {
	return schema.ClustersSchema.OptionsMap
}
