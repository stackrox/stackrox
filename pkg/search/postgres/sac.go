package postgres

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// enrichQueryWithSACFilter is used in standardizeQueryAndPopulatePath.
// The role of the enrichQueryWithSACFilter function is to ensure data is
// filtered according to the requested access scope.
func enrichQueryWithSACFilter(ctx context.Context, q *v1.Query, schema *walker.Schema, queryType QueryType) (*v1.Query, error) {
	switch queryType {
	// DELETE is expected to be the only Write use case for the query generator
	case DELETE:
		if schema.PermissionChecker != nil {
			if ok, err := schema.PermissionChecker.WriteAllowed(ctx); err != nil {
				return nil, err
			} else if !ok {
				return nil, sac.ErrResourceAccessDenied
			}
			return q, nil
		}
		sacFilter, err := GetReadWriteSACQuery(ctx, schema.ScopingResource)
		if err != nil {
			return nil, err
		}
		if isEmptySACFilter(sacFilter) {
			return q, nil
		}
		pagination := q.GetPagination()
		query := searchPkg.ConjunctionQuery(sacFilter, q)
		query.Pagination = pagination
		return query, nil
	default:
		if schema.PermissionChecker != nil {
			if ok, err := schema.PermissionChecker.ReadAllowed(ctx); err != nil {
				return nil, err
			} else if !ok {
				return getMatchNoneQuery(), nil
			}
			return q, nil
		}
		sacFilter, err := GetReadSACQuery(ctx, schema.ScopingResource)
		if err != nil {
			return nil, err
		}
		if isEmptySACFilter(sacFilter) {
			return q, nil
		}
		pagination := q.GetPagination()
		query := searchPkg.ConjunctionQuery(sacFilter, q)
		query.Pagination = pagination
		return query, nil
	}
}

func isEmptySACFilter(sacFilter *v1.Query) bool {
	// Hack to avoid having non-nil query but nil queryEntry in standardizeQueryAndPopulatePath
	// which then results in Walk with unrestricted scope failing
	switch sacFilter.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		return false
	case *v1.Query_Conjunction:
		return false
	case *v1.Query_Disjunction:
		return false
	case *v1.Query_BooleanQuery:
		return false
	}
	return true
}

// GetReadWriteSACQuery returns SAC filter for resource or error is permission is denied.
func GetReadWriteSACQuery(ctx context.Context, targetResource permissions.ResourceMetadata) (*v1.Query, error) {
	return getSACQuery(ctx, targetResource, storage.Access_READ_WRITE_ACCESS)
}

// GetReadSACQuery returns SAC filter for resource or error is permission is denied.
func GetReadSACQuery(ctx context.Context, targetResource permissions.ResourceMetadata) (*v1.Query, error) {
	sacQuery, err := getSACQuery(ctx, targetResource, storage.Access_READ_ACCESS)
	if errors.Is(err, sac.ErrResourceAccessDenied) {
		return getMatchNoneQuery(), nil
	}
	if targetResource.String() == resources.WorkflowAdministration.String() {
		log.Infof("For workflow administration, got the following SAC READ query: %+v", sacQuery)
	}
	return sacQuery, err
}

func getSACQuery(ctx context.Context, targetResource permissions.ResourceMetadata, access storage.Access) (*v1.Query, error) {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(access).Resource(targetResource)
	action := permissions.View
	if access == storage.Access_READ_WRITE_ACCESS {
		action = permissions.Modify
	}
	switch targetResource.GetScope() {
	case permissions.GlobalScope:
		id := authn.IdentityFromContextOrNil(ctx)
		// (dhaus): Workaround; only attempt to filter by teams if the resource is team scoped; and we actually have
		// teams in the current identity context. If not we just simply skip this and treat this as "global".
		//
		// For implementing later, probably would be good to:
		// - add a new "scope" that is "lower" than global scope to reduce the edge cases within a single scope.
		// - add the information about which teams one "effectively" has access to by including it into the EAS;
		// 	 this might be controversial since we are only ever really adding the same list associated with the identity.
		//   We probably then have to also check whether we need to change the EAS state, but EAS state _really_ only is
		//   for cluster / namespace right now. Might be time for a new "abstraction" which is separate from "traditional"
		//   EAS solely for team specific stuff.
		// - IsAllowed() shouldn't be called but rather should be done within the EAS state calculation / whatever abstraction
		//   will be used.
		if targetResource.GetTeamScope() && id != nil && id.Teams() != nil {
			log.Info("Found a resource with team scope AND we have teams associated with the current identity")
			teamNames := getTeamNames(id.Teams())
			log.Infof("Found the following team names: %+v", teamNames)
			if !scopeChecker.IsAllowed(sac.TeamScopeKeys(teamNames...)...) {
				log.Infof("Not allowed with team names %+v for resource %s", teamNames, targetResource.String())
				return nil, sac.ErrResourceAccessDenied
			}
			log.Info("Building SAC Team Level Query filters...")
			return sac.BuildTeamLevelSACQueryFilter(teamNames)
		}

		if !scopeChecker.IsAllowed() {
			return nil, sac.ErrResourceAccessDenied
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

func getTeamNames(teams []*storage.Team) []string {
	teamNames := make([]string, 0, len(teams))
	for _, team := range teams {
		teamNames = append(teamNames, team.GetName())
	}
	return teamNames
}
