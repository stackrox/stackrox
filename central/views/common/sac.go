package common

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

// WithSACFilter decorates the given query with the scope query. This scope query will filter out target resources with READ access
// in the given context's access scope
func WithSACFilter(ctx context.Context, targetResource permissions.ResourceMetadata, query *v1.Query) (*v1.Query, error) {
	var sacQueryFilter *v1.Query

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return nil, err
	}

	if clusterLevelResource(targetResource) {
		sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
	} else {
		sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
	}
	if err != nil {
		return nil, err
	}

	return search.FilterQueryByQuery(query, sacQueryFilter), nil
}

func clusterLevelResource(targetResource permissions.ResourceMetadata) bool {
	return targetResource == resources.Cluster || targetResource == resources.Node
}
