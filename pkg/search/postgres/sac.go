package postgres

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
)

// GetReadWriteSACQuery returns SAC filter for resource or error is permission is denied.
func GetReadWriteSACQuery(ctx context.Context, targetResource permissions.ResourceMetadata) (*v1.Query, error) {
	return getSACQuery(ctx, targetResource, storage.Access_READ_WRITE_ACCESS)
}

// GetReadSACQuery returns SAC filter for resource or error is permission is denied.
func GetReadSACQuery(ctx context.Context, targetResource permissions.ResourceMetadata) (*v1.Query, error) {
	return getSACQuery(ctx, targetResource, storage.Access_READ_ACCESS)
}

func getSACQuery(ctx context.Context, targetResource permissions.ResourceMetadata, access storage.Access) (*v1.Query, error) {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(access).Resource(targetResource)
	action := permissions.View
	if access == storage.Access_READ_WRITE_ACCESS {
		action = permissions.Modify
	}
	switch targetResource.GetScope() {
	case permissions.GlobalScope:
		if !scopeChecker.IsAllowed() {
			return getMatchNoneQuery(), nil
		}
		return &v1.Query{}, nil
	case permissions.ClusterScope:
		scopeTree, err := scopeChecker.EffectiveAccessScope(action(targetResource))
		if err != nil {
			return nil, err
		}
		return sac.BuildNonVerboseClusterLevelSACQueryFilter(scopeTree)
	case permissions.NamespaceScope:
		scopeTree, err := scopeChecker.EffectiveAccessScope(action(targetResource))
		if err != nil {
			return nil, err
		}
		return sac.BuildNonVerboseClusterNamespaceLevelSACQueryFilter(scopeTree)
	}
	return nil, fmt.Errorf("could not prepare SAC Query for %s", targetResource)
}

func getMatchNoneQuery() *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchNoneQuery{
					MatchNoneQuery: &v1.MatchNoneQuery{},
				},
			},
		},
	}
}
